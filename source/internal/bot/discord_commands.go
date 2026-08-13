package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	discordInteractionEphemeral = 1 << 6
	discordGatewayVersion       = "10"
	discordGuildsIntent         = 1 << 0
)

type discordApplication struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Bot      bool   `json:"bot"`
}

type discordGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Unavailable bool   `json:"unavailable"`
}

type discordInteraction struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Type      int    `json:"type"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
	Member    *struct {
		Permissions string `json:"permissions"`
	} `json:"member"`
	Channel *struct {
		Name string `json:"name"`
	} `json:"channel"`
	Guild *discordGuild `json:"guild"`
	Data  struct {
		Name    string `json:"name"`
		Options []struct {
			Type int    `json:"type"`
			Name string `json:"name"`
		} `json:"options"`
	} `json:"data"`
}

type discordReply struct {
	Content string `json:"content"`
	Flags   int    `json:"flags,omitempty"`
}

type discordGatewayPacket struct {
	Op       int             `json:"op"`
	Data     json.RawMessage `json:"d"`
	Sequence *int64          `json:"s"`
	Event    string          `json:"t"`
}

func (s *Sender) WithDNS(server string) *Sender {
	server = strings.TrimSpace(server)
	if server == s.DNSServer {
		return s
	}
	copy := *s
	copy.DNSServer = server
	return &copy
}

func (s *Sender) netDialer() *net.Dialer {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	if s.DNSServer == "" {
		return dialer
	}
	resolverDialer := &net.Dialer{Timeout: 5 * time.Second}
	dnsAddress := s.DNSServer
	if _, _, err := net.SplitHostPort(dnsAddress); err != nil {
		dnsAddress = net.JoinHostPort(dnsAddress, "53")
	}
	dialer.Resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return resolverDialer.DialContext(ctx, network, dnsAddress)
		},
	}
	return dialer
}

func (s *Sender) httpClient() *http.Client {
	if s.DNSServer == "" {
		return s.Client
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if configured, ok := s.Client.Transport.(*http.Transport); ok {
		transport = configured.Clone()
	}
	transport.DialContext = s.netDialer().DialContext
	client := *s.Client
	client.Transport = transport
	return &client
}

func (s *Sender) discordGatewayDialer() *websocket.Dialer {
	dialer := *websocket.DefaultDialer
	if s.DNSServer != "" {
		dialer.NetDialContext = s.netDialer().DialContext
	}
	return &dialer
}

func (s *Sender) discordJSON(ctx context.Context, method, path, token string, input, output any) error {
	var encoded []byte
	if input != nil {
		var err error
		encoded, err = json.Marshal(input)
		if err != nil {
			return err
		}
	}
	bases := []string{s.DiscordBaseURL}
	if fallback := strings.TrimSpace(s.DiscordFallbackBaseURL); fallback != "" && strings.TrimRight(fallback, "/") != strings.TrimRight(s.DiscordBaseURL, "/") {
		bases = append(bases, fallback)
	}
	client := s.httpClient()
	if s.DNSServer != "" {
		defer client.CloseIdleConnections()
	}
	for index, base := range bases {
		endpoint := strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
		var body io.Reader
		if input != nil {
			body = bytes.NewReader(encoded)
		}
		request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
		if err != nil {
			return errors.New("Discord endpoint is invalid")
		}
		request.Header.Set("User-Agent", "Bonghos/notification-bot")
		if token != "" {
			request.Header.Set("Authorization", "Bot "+token)
		}
		if input != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		response, err := client.Do(request)
		if err != nil {
			if index+1 < len(bases) && ctx.Err() == nil && discordFallbackAllowed(method, err) {
				continue
			}
			return discordNetworkError("Discord request", endpoint, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		response.Body.Close()
		if readErr != nil {
			return errors.New("reading Discord response failed")
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("Discord returned HTTP %d", response.StatusCode)
		}
		if output != nil && len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, output); err != nil {
				return errors.New("Discord returned invalid response data")
			}
		}
		return nil
	}
	return errors.New("Discord request failed")
}

func discordFallbackAllowed(method string, requestErr error) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	// A DNS failure happens before a non-idempotent request can reach Discord.
	// Retrying other POST transport failures could duplicate a delivered message
	// or interaction response whose acknowledgement was lost.
	var dnsErr *net.DNSError
	return errors.As(requestErr, &dnsErr)
}

func discordNetworkError(operation, endpoint string, requestErr error) error {
	host := "Discord"
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Hostname() != "" {
		host = parsed.Hostname()
	}
	var dnsErr *net.DNSError
	if errors.As(requestErr, &dnsErr) {
		return fmt.Errorf("%s failed: DNS lookup failed for %s", operation, host)
	}
	var networkErr net.Error
	if errors.As(requestErr, &networkErr) && networkErr.Timeout() {
		return fmt.Errorf("%s failed: timeout contacting %s", operation, host)
	}
	return fmt.Errorf("%s failed: network error contacting %s", operation, host)
}

func (s *Sender) resolveDiscordApplication(ctx context.Context, token string) (discordApplication, error) {
	var application discordApplication
	if err := s.discordJSON(ctx, http.MethodGet, "users/@me", token, nil, &application); err != nil {
		return discordApplication{}, err
	}
	if !discordDestinationRE.MatchString(application.ID) || !application.Bot {
		return discordApplication{}, errors.New("Discord token does not belong to a bot application")
	}
	return application, nil
}

func discordCommandDefinition() map[string]any {
	option := func(name, description string) map[string]any {
		return map[string]any{"type": 1, "name": name, "description": description}
	}
	return map[string]any{
		"name": "bonghos", "description": "Configure Bonghos notifications for this channel",
		"type": 1, "dm_permission": false, "default_member_permissions": "32",
		"options": []map[string]any{
			option("here", "Send Bonghos notifications to this channel"),
			option("where", "Check this channel's Bonghos notification status"),
			option("disconnect", "Stop Bonghos notifications in this channel"),
			option("help", "Show Bonghos notification commands"),
		},
	}
}

func (s *Sender) registerDiscordCommands(ctx context.Context, token, applicationID, guildID string) error {
	if !discordDestinationRE.MatchString(applicationID) {
		return errors.New("Discord application ID is invalid")
	}
	path := "applications/" + url.PathEscape(applicationID)
	if guildID != "" {
		if !discordDestinationRE.MatchString(guildID) {
			return errors.New("Discord guild ID is invalid")
		}
		path += "/guilds/" + url.PathEscape(guildID)
	}
	path += "/commands"
	return s.discordJSON(ctx, http.MethodPut, path, token, []any{discordCommandDefinition()}, nil)
}

func discordSubcommand(interaction discordInteraction) string {
	for _, option := range interaction.Data.Options {
		if option.Type == 1 {
			name := strings.ToLower(option.Name)
			if name == "here" || name == "where" || name == "disconnect" || name == "help" {
				return name
			}
		}
	}
	return "help"
}

func discordAdministrator(interaction discordInteraction) bool {
	if interaction.Member == nil {
		return false
	}
	permissions, err := strconv.ParseUint(interaction.Member.Permissions, 10, 64)
	if err != nil {
		return false
	}
	const administrator = uint64(1 << 3)
	const manageGuild = uint64(1 << 5)
	return permissions&administrator != 0 || permissions&manageGuild != 0
}

func (d *Dispatcher) handleDiscordInteraction(state DiscordCommandState, interaction discordInteraction) *discordReply {
	if interaction.Type != 2 || interaction.Data.Name != "bonghos" {
		return nil
	}
	reply := func(content string) *discordReply {
		return &discordReply{Content: content, Flags: discordInteractionEphemeral}
	}
	if interaction.GuildID == "" || !discordDestinationRE.MatchString(interaction.ChannelID) {
		return reply("Bonghos notifications can only be configured inside a server channel.")
	}
	command := discordSubcommand(interaction)
	if !discordAdministrator(interaction) {
		return reply("Only a server administrator can use Bonghos configuration commands.")
	}
	if command == "help" {
		return reply("Bonghos commands:\n`/bonghos here`: Send notifications to this channel\n`/bonghos where`: Check this channel's status\n`/bonghos disconnect`: Stop notifications here")
	}
	switch command {
	case "here":
		name := "Channel " + interaction.ChannelID
		if interaction.Channel != nil && strings.TrimSpace(interaction.Channel.Name) != "" {
			name = strings.TrimSpace(interaction.Channel.Name)
		}
		destination := Destination{ID: interaction.ChannelID, Name: name, Type: "channel", GuildID: interaction.GuildID}
		if interaction.Guild != nil {
			destination.GuildName = strings.TrimSpace(interaction.Guild.Name)
			destination.GuildIcon = strings.TrimSpace(interaction.Guild.Icon)
		}
		err := d.Store.SetDiscordDestination(state.BotID, destination)
		if err != nil {
			if strings.Contains(err.Error(), "three connected channels") {
				return reply("Bonghos already has three connected Discord channels.")
			}
			d.logCommand("connecting a Discord destination for %s: %v", state.BotName, err)
			return reply("Bonghos could not connect this channel. Check the Bonghos logs for details.")
		}
		return reply("Bonghos notifications will be sent to this channel.")
	case "disconnect":
		if err := d.Store.DisconnectDiscordDestination(state.BotID, interaction.ChannelID); err != nil {
			return reply("Bonghos could not disconnect this channel.")
		}
		return reply("Bonghos notifications are disconnected from this channel.")
	case "where":
		_, err := d.Store.DiscordDestination(state.BotID, interaction.ChannelID)
		if errors.Is(err, ErrNotFound) {
			return reply("This channel is not connected. Run `/bonghos here` in the channel that should receive notifications.")
		}
		if err != nil {
			return reply("Bonghos could not read this destination.")
		}
		return reply("Bonghos notifications are configured for this channel.")
	default:
		return reply("Use `/bonghos help` to list available commands.")
	}
}

func (s *Sender) respondDiscordInteraction(ctx context.Context, interaction discordInteraction, reply discordReply) error {
	if !discordDestinationRE.MatchString(interaction.ID) || interaction.Token == "" {
		return errors.New("Discord interaction response is invalid")
	}
	path := "interactions/" + url.PathEscape(interaction.ID) + "/" + url.PathEscape(interaction.Token) + "/callback"
	return s.discordJSON(ctx, http.MethodPost, path, "", map[string]any{"type": 4, "data": reply}, nil)
}

type discordWorker struct {
	token     string
	dnsServer string
	cancel    context.CancelFunc
}

// RunDiscordCommands manages a Gateway worker for every configured Discord
// bot. Store changes are reconciled without restarting the control plane.
func (d *Dispatcher) RunDiscordCommands(ctx context.Context) {
	if d == nil || d.Store == nil || d.Sender == nil {
		return
	}
	workers := make(map[int64]discordWorker)
	reconcile := func() {
		states, err := d.Store.DiscordCommandBots()
		if err != nil {
			d.logCommand("listing Discord command bots: %v", err)
			return
		}
		desired := make(map[int64]DiscordCommandState, len(states))
		for _, state := range states {
			desired[state.BotID] = state
			if current, ok := workers[state.BotID]; ok && current.token == state.Token && current.dnsServer == state.DNSServer {
				continue
			} else if ok {
				current.cancel()
			}
			workerCtx, cancel := context.WithCancel(ctx)
			workers[state.BotID] = discordWorker{token: state.Token, dnsServer: state.DNSServer, cancel: cancel}
			go d.runDiscordWorker(workerCtx, state)
		}
		for id, worker := range workers {
			if _, ok := desired[id]; !ok {
				worker.cancel()
				delete(workers, id)
			}
		}
	}
	reconcile()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	defer func() {
		for _, worker := range workers {
			worker.cancel()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcile()
		}
	}
}

func (d *Dispatcher) runDiscordWorker(ctx context.Context, state DiscordCommandState) {
	sender := d.Sender.WithDNS(state.DNSServer)
	backoff := time.Second
	var application discordApplication
	globalRegistered := false
	for ctx.Err() == nil {
		if application.ID == "" {
			resolved, err := sender.resolveDiscordApplication(ctx, state.Token)
			if err != nil {
				d.logCommand("Discord setup for %s failed: %v", state.BotName, err)
			} else {
				application = resolved
				backoff = time.Second
			}
		}
		if application.ID != "" {
			if !globalRegistered {
				registerCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
				if err := sender.registerDiscordCommands(registerCtx, state.Token, application.ID, ""); err == nil {
					globalRegistered = true
				} else {
					d.logCommand("Discord global command registration for %s failed: %v", state.BotName, err)
				}
				cancel()
			}
			if err := d.runDiscordGateway(ctx, state, application.ID); err != nil && ctx.Err() == nil {
				d.logCommand("Discord gateway for %s: %v", state.BotName, err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (d *Dispatcher) runDiscordGateway(ctx context.Context, state DiscordCommandState, applicationID string) error {
	sender := d.Sender.WithDNS(state.DNSServer)
	gateway := strings.TrimRight(sender.DiscordGatewayURL, "/") + "/?v=" + discordGatewayVersion + "&encoding=json"
	conn, _, err := sender.discordGatewayDialer().DialContext(ctx, gateway, http.Header{"User-Agent": []string{"Bonghos/notification-bot"}})
	if err != nil {
		return discordNetworkError("Discord gateway connection", gateway, err)
	}
	defer conn.Close()

	packets := make(chan discordGatewayPacket)
	readErrors := make(chan error, 1)
	go func() {
		for {
			var packet discordGatewayPacket
			if err := conn.ReadJSON(&packet); err != nil {
				readErrors <- err
				return
			}
			select {
			case packets <- packet:
			case <-ctx.Done():
				return
			}
		}
	}()
	var sequence int64
	heartbeatAcknowledged := true
	var heartbeat *time.Timer
	var heartbeatC <-chan time.Time
	resetHeartbeat := func(delay time.Duration) {
		if heartbeat == nil {
			heartbeat = time.NewTimer(delay)
		} else {
			if !heartbeat.Stop() {
				select {
				case <-heartbeat.C:
				default:
				}
			}
			heartbeat.Reset(delay)
		}
		heartbeatC = heartbeat.C
	}
	defer func() {
		if heartbeat != nil {
			heartbeat.Stop()
		}
	}()
	var heartbeatInterval time.Duration
	registered := make(map[string]bool)
	guilds := make(map[string]discordGuild)
	registerGuild := func(guildID string) {
		if !discordDestinationRE.MatchString(guildID) || registered[guildID] {
			return
		}
		registered[guildID] = true
		go func() {
			registerCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			if err := sender.registerDiscordCommands(registerCtx, state.Token, applicationID, guildID); err != nil {
				d.logCommand("Discord guild command registration for %s failed: %v", state.BotName, err)
			}
		}()
	}
	write := func(value any) error {
		if err := conn.WriteJSON(value); err != nil {
			return errors.New("Discord gateway write failed")
		}
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Bonghos stopped"), time.Now().Add(time.Second))
			return nil
		case <-heartbeatC:
			if !heartbeatAcknowledged {
				return errors.New("Discord gateway heartbeat timed out")
			}
			heartbeatAcknowledged = false
			if err := write(map[string]any{"op": 1, "d": sequence}); err != nil {
				return err
			}
			resetHeartbeat(heartbeatInterval)
		case <-readErrors:
			return errors.New("Discord gateway disconnected")
		case packet := <-packets:
			if packet.Sequence != nil {
				sequence = *packet.Sequence
			}
			switch packet.Op {
			case 10:
				var hello struct {
					HeartbeatInterval int64 `json:"heartbeat_interval"`
				}
				if json.Unmarshal(packet.Data, &hello) != nil || hello.HeartbeatInterval <= 0 {
					return errors.New("Discord gateway returned invalid hello")
				}
				heartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond
				resetHeartbeat(heartbeatInterval / 2)
				if err := write(map[string]any{"op": 2, "d": map[string]any{
					"token": state.Token, "intents": discordGuildsIntent,
					"properties": map[string]string{"os": "linux", "browser": "bonghos", "device": "bonghos"},
				}}); err != nil {
					return err
				}
			case 1:
				if err := write(map[string]any{"op": 1, "d": sequence}); err != nil {
					return err
				}
			case 11:
				heartbeatAcknowledged = true
			case 7, 9:
				return errors.New("Discord gateway requested reconnect")
			case 0:
				switch packet.Event {
				case "READY":
					var ready struct {
						Guilds []struct {
							ID string `json:"id"`
						} `json:"guilds"`
					}
					if json.Unmarshal(packet.Data, &ready) == nil {
						for _, guild := range ready.Guilds {
							registerGuild(guild.ID)
						}
					}
				case "GUILD_CREATE":
					var guild discordGuild
					if json.Unmarshal(packet.Data, &guild) == nil {
						guilds[guild.ID] = guild
						registerGuild(guild.ID)
						if !guild.Unavailable {
							_, err := d.Store.MergeDiscovered(state.BotID, []Destination{{
								ID: guild.ID, Name: strings.TrimSpace(guild.Name), Type: "guild",
								GuildID: guild.ID, GuildName: strings.TrimSpace(guild.Name), GuildIcon: strings.TrimSpace(guild.Icon),
							}})
							if err != nil {
								d.logCommand("remembering Discord server for %s: %v", state.BotName, err)
							}
						}
					}
				case "GUILD_DELETE":
					var guild discordGuild
					if json.Unmarshal(packet.Data, &guild) == nil && !guild.Unavailable {
						delete(guilds, guild.ID)
						if err := d.Store.ForgetDiscovered(state.BotID, guild.ID); err != nil {
							d.logCommand("forgetting Discord server for %s: %v", state.BotName, err)
						}
					}
				case "INTERACTION_CREATE":
					var interaction discordInteraction
					if json.Unmarshal(packet.Data, &interaction) != nil {
						continue
					}
					if guild, ok := guilds[interaction.GuildID]; ok {
						interaction.Guild = &guild
					}
					reply := d.handleDiscordInteraction(state, interaction)
					if reply != nil {
						go func() {
							responseCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
							defer cancel()
							if err := sender.respondDiscordInteraction(responseCtx, interaction, *reply); err != nil {
								d.logCommand("Discord interaction reply for %s failed: %v", state.BotName, err)
							}
						}()
					}
				}
			}
		}
	}
}
