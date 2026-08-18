package playit

import (
	"encoding/json"
	"net"
	"testing"
)

func ipcStatusForTest(t *testing.T, lifecycle any) AgentStatus {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	go func() {
		defer server.Close()
		encoder := json.NewEncoder(server)
		decoder := json.NewDecoder(server)
		_ = encoder.Encode(map[string]any{
			"message_kind": "hello",
			"data":         map[string]any{"protocol": map[string]any{"ipc_version": playitIPCVersion, "capabilities": []string{}}},
		})
		var request map[string]any
		_ = decoder.Decode(&request)
		_ = encoder.Encode(map[string]any{
			"message_kind": "response",
			"data": map[string]any{
				"ipc_version": playitIPCVersion,
				"request_id":  1,
				"response":    map[string]any{"type": "state", "data": lifecycle},
			},
		})
	}()
	status, err := queryAgentStatus(client)
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func TestIPCRequiresRunningLifecycle(t *testing.T) {
	starting := ipcStatusForTest(t, map[string]any{"state": "starting"})
	if starting.Running || starting.Phase != "starting" {
		t.Fatalf("starting status = %+v", starting)
	}
	running := ipcStatusForTest(t, map[string]any{"state": "running", "data": map[string]any{
		"version": "1.0.10", "agent_id": "agent-id",
	}})
	if !running.Running || running.AgentID != "agent-id" || running.Version != "1.0.10" {
		t.Fatalf("running status = %+v", running)
	}
}

func TestIPCReportsSafeLifecycleError(t *testing.T) {
	status := ipcStatusForTest(t, map[string]any{"state": "has_invalid_secret", "data": map[string]any{
		"code": "invalid_secret", "message": "Agent secret was rejected", "retryable": false,
	}})
	if status.Running || status.Phase != "has_invalid_secret" || status.Error != "The Playit agent rejected its credential; relink the agent" {
		t.Fatalf("error status = %+v", status)
	}
}
