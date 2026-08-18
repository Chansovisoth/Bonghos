package app

import (
	"testing"

	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

func TestWebsocketTopicAuthorization(t *testing.T) {
	tests := []struct {
		role    authorization.Role
		topic   string
		allowed bool
	}{
		{authorization.RoleViewer, "overview", true},
		{authorization.RoleViewer, "overview_performance", false},
		{authorization.RoleViewer, "performance", false},
		{authorization.RoleViewer, "console", true},
		{authorization.RoleViewer, "console_use", false},
		{authorization.RoleViewer, "activity", false},
		{authorization.RoleMember, "players", true},
		{authorization.RoleMember, "performance", false},
		{authorization.RoleMember, "backups", false},
		{authorization.RoleAdmin, "backups", true},
		{authorization.RoleAdmin, "schedules", true},
		{authorization.RoleAdmin, "activity", true},
		{authorization.RoleAdmin, "performance", true},
		{authorization.RoleAdmin, "overview_performance", true},
		{authorization.RoleOwner, "activity", true},
		{authorization.RoleOwner, "unknown", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.role)+"/"+tt.topic, func(t *testing.T) {
			if got := websocketTopicAllowed(tt.role, tt.topic); got != tt.allowed {
				t.Fatalf("websocketTopicAllowed(%q, %q) = %v, want %v", tt.role, tt.topic, got, tt.allowed)
			}
		})
	}
}
