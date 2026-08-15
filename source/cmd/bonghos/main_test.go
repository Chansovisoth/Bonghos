package main

import (
	"reflect"
	"testing"
)

func TestCommandAndArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "no command starts the background web panel",
			wantCmd:  "web",
			wantArgs: []string{"start"},
		},
		{
			name:     "explicit command is unchanged",
			input:    []string{"serve"},
			wantCmd:  "serve",
			wantArgs: []string{},
		},
		{
			name:     "explicit arguments are preserved",
			input:    []string{"web", "logs", "--follow"},
			wantCmd:  "web",
			wantArgs: []string{"logs", "--follow"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotArgs := commandAndArgs(tt.input)
			if gotCmd != tt.wantCmd || !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Fatalf("commandAndArgs(%v) = %q, %v; want %q, %v", tt.input, gotCmd, gotArgs, tt.wantCmd, tt.wantArgs)
			}
		})
	}
}
