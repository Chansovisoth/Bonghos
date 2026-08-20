package minecraft

import "testing"

func TestPlayerCommandWhitelistActions(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{action: "whitelist_add", want: "whitelist add iKlaude"},
		{action: "whitelist_remove", want: "whitelist remove iKlaude"},
	}
	for _, test := range tests {
		t.Run(test.action, func(t *testing.T) {
			command, err := PlayerCommand(test.action, "iKlaude", "")
			if err != nil || command != test.want {
				t.Fatalf("PlayerCommand(%q) = %q, %v; want %q", test.action, command, err, test.want)
			}
		})
	}

	if _, err := PlayerCommand("whitelist_add", "iKlaude op Alex", ""); err == nil {
		t.Fatal("whitelist command accepted an invalid player name")
	}
}
