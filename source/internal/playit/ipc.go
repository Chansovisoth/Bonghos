package playit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"
)

const playitIPCVersion = 2

var errUnsupportedIPC = errors.New("unsupported Playit IPC protocol")

// AgentStatus is the daemon's own lifecycle state. A running process is not
// considered online until playitd reports the Running lifecycle over IPC.
type AgentStatus struct {
	Phase   string `json:"agent_phase"`
	Running bool   `json:"agent_online"`
	AgentID string `json:"agent_id,omitempty"`
	Version string `json:"agent_version,omitempty"`
	Error   string `json:"agent_error,omitempty"`
}

func AgentSocketPath(home string) string {
	return filepath.Join(home, "system", "runtime", "playit", "playitd.sock")
}

// ManagedAgentStatus reads the private socket created for Bonghos's playitd.
// Failure to connect means stopped/starting; it never falls back to process
// detection because that cannot prove agent registration or compatibility.
func ManagedAgentStatus(ctx context.Context, home string) AgentStatus {
	queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(queryCtx, "unix", AgentSocketPath(home))
	if err != nil {
		return AgentStatus{Phase: "stopped"}
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	status, err := queryAgentStatus(conn)
	if err != nil {
		if errors.Is(err, errUnsupportedIPC) {
			return AgentStatus{Phase: "incompatible", Error: "The installed Playit agent does not support status checks; update playitd"}
		}
		return AgentStatus{Phase: "starting"}
	}
	return status
}

func queryAgentStatus(conn io.ReadWriter) (AgentStatus, error) {
	decoder := json.NewDecoder(io.LimitReader(conn, 1<<20))
	encoder := json.NewEncoder(conn)
	var hello struct {
		Kind string `json:"message_kind"`
		Data struct {
			Protocol struct {
				Version int `json:"ipc_version"`
			} `json:"protocol"`
		} `json:"data"`
	}
	if err := decoder.Decode(&hello); err != nil {
		return AgentStatus{}, err
	}
	if hello.Kind != "hello" || hello.Data.Protocol.Version != playitIPCVersion {
		return AgentStatus{}, errUnsupportedIPC
	}
	if err := encoder.Encode(map[string]any{
		"ipc_version": playitIPCVersion,
		"request_id":  1,
		"request":     map[string]string{"type": "get_state"},
	}); err != nil {
		return AgentStatus{}, err
	}
	for {
		var envelope struct {
			Kind string `json:"message_kind"`
			Data struct {
				Version   int             `json:"ipc_version"`
				RequestID int             `json:"request_id"`
				Response  json.RawMessage `json:"response"`
			} `json:"data"`
		}
		if err := decoder.Decode(&envelope); err != nil {
			return AgentStatus{}, err
		}
		if envelope.Kind != "response" || envelope.Data.RequestID != 1 {
			continue
		}
		if envelope.Data.Version != playitIPCVersion {
			return AgentStatus{}, errUnsupportedIPC
		}
		var response struct {
			Type string `json:"type"`
			Data struct {
				State string `json:"state"`
				Data  struct {
					Version string `json:"version"`
					AgentID string `json:"agent_id"`
				} `json:"data"`
			} `json:"data"`
		}
		if err := json.Unmarshal(envelope.Data.Response, &response); err != nil || response.Type != "state" {
			return AgentStatus{}, errUnsupportedIPC
		}
		phase := strings.TrimSpace(response.Data.State)
		status := AgentStatus{Phase: phase, AgentID: response.Data.Data.AgentID, Version: response.Data.Data.Version}
		status.Running = phase == "running"
		switch phase {
		case "has_invalid_secret":
			status.Error = "The Playit agent rejected its credential; relink the agent"
		case "disabled_over_limit":
			status.Error = "The Playit account has reached its agent limit"
		case "error":
			status.Error = "The Playit agent could not start; check its service logs"
		}
		return status, nil
	}
}
