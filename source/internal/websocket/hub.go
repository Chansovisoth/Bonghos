// Package websocket implements the authenticated hub. Connections exist only
// while authenticated pages are open; everything important continues without
// them. Topics map to pages: overview, console, performance, players,
// servers, backups, schedules.
package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

const (
	maxMessageBytes = 8 * 1024
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = 45 * time.Second
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Same-origin check: the embedded UI is served from the same host.
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return origin == "http://"+r.Host || origin == "https://"+r.Host
	},
}

// Event is a broadcast message.
type Event struct {
	Topic string          `json:"topic"`
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data"`
}

type client struct {
	conn           *ws.Conn
	send           chan []byte
	topics         map[string]bool
	topicIntervals map[string]time.Duration
	lastTopicSent  map[string]time.Time
	userID         int64
	canUse         func(topic string) bool // per-role topic authorization
	mu             sync.Mutex
	onInput        func(topic string, data json.RawMessage)
}

// Hub tracks connected clients and their subscriptions.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]bool
	// OnConsoleCommand is invoked for console input events (authorized upstream).
	OnConsoleCommand func(userID int64, command string)
}

func NewHub() *Hub {
	return &Hub{clients: map[*client]bool{}}
}

// SubscriberCount returns how many clients subscribe to a topic (used to
// avoid unnecessary polling like periodic `list` when nobody is watching).
func (h *Hub) SubscriberCount(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	n := 0
	for c := range h.clients {
		c.mu.Lock()
		if c.topics[topic] {
			n++
		}
		c.mu.Unlock()
	}
	return n
}

// MinimumInterval returns the fastest requested delivery interval for a topic.
// A zero interval means the client chose the server default.
func (h *Hub) MinimumInterval(topic string, fallback time.Duration) time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	minimum := fallback
	for c := range h.clients {
		c.mu.Lock()
		subscribed := c.topics[topic]
		interval := c.topicIntervals[topic]
		c.mu.Unlock()
		if subscribed && interval > 0 && interval < minimum {
			minimum = interval
		}
	}
	return minimum
}

// Broadcast sends an event to all clients subscribed to its topic.
func (h *Hub) Broadcast(topic, typ string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg, _ := json.Marshal(Event{Topic: topic, Type: typ, Data: raw})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		c.mu.Lock()
		subscribed := c.topics[topic]
		c.mu.Unlock()
		if !subscribed {
			continue
		}
		select {
		case c.send <- msg:
		default: // slow client: drop message rather than block
		}
	}
}

// BroadcastDue sends a topic event according to each subscriber's requested
// cadence. Subscribers that did not request one receive the server default.
func (h *Hub) BroadcastDue(topic, typ string, data any, fallback time.Duration) {
	h.broadcastDueAt(topic, typ, data, fallback, time.Now())
}

func (h *Hub) broadcastDueAt(topic, typ string, data any, fallback time.Duration, now time.Time) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg, _ := json.Marshal(Event{Topic: topic, Type: typ, Data: raw})
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		c.mu.Lock()
		subscribed := c.topics[topic]
		interval := c.topicIntervals[topic]
		if interval <= 0 {
			interval = fallback
		}
		lastSent := c.lastTopicSent[topic]
		due := subscribed && (lastSent.IsZero() || now.Sub(lastSent) >= interval)
		if due {
			c.lastTopicSent[topic] = now
		}
		c.mu.Unlock()
		if !due {
			continue
		}
		select {
		case c.send <- msg:
		default: // slow client: drop message rather than block
		}
	}
}

// Serve upgrades an already-authenticated request. canUse gates topics by
// the user's role (e.g. Members cannot subscribe to console).
func (h *Hub) Serve(w http.ResponseWriter, r *http.Request, userID int64,
	canUse func(topic string) bool, onCommand func(command string)) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{
		conn: conn, send: make(chan []byte, 256),
		topics: map[string]bool{}, topicIntervals: map[string]time.Duration{},
		lastTopicSent: map[string]time.Time{}, userID: userID, canUse: canUse,
	}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	go c.writeLoop()
	c.readLoop(h, onCommand)

	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.send)
}

func (c *client) readLoop(h *Hub, onCommand func(string)) {
	defer c.conn.Close()
	c.conn.SetReadLimit(maxMessageBytes)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Action          string `json:"action"` // subscribe | unsubscribe | console_command
			Type            string `json:"type"`   // accepted alias for "action"
			Topic           string `json:"topic"`
			Command         string `json:"command"`
			IntervalSeconds int    `json:"interval_seconds"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		// Accept either key. "action" is canonical, but tolerating "type"
		// means a client using the other spelling degrades to nothing
		// happening rather than silently receiving no events at all.
		verb := msg.Action
		if verb == "" {
			verb = msg.Type
		}
		switch verb {
		case "subscribe":
			if c.canUse == nil || c.canUse(msg.Topic) {
				interval := time.Duration(0)
				if msg.Topic == "performance" && msg.IntervalSeconds > 0 {
					seconds := msg.IntervalSeconds
					if seconds < 1 {
						seconds = 1
					}
					if seconds > 60 {
						seconds = 60
					}
					interval = time.Duration(seconds) * time.Second
				}
				c.mu.Lock()
				c.topics[msg.Topic] = true
				c.topicIntervals[msg.Topic] = interval
				delete(c.lastTopicSent, msg.Topic)
				c.mu.Unlock()
			}
		case "unsubscribe":
			c.mu.Lock()
			delete(c.topics, msg.Topic)
			delete(c.topicIntervals, msg.Topic)
			delete(c.lastTopicSent, msg.Topic)
			c.mu.Unlock()
		case "console_command":
			if (c.canUse == nil || c.canUse("console_use")) && onCommand != nil {
				onCommand(msg.Command)
			}
		}
	}
}

func (c *client) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(ws.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
