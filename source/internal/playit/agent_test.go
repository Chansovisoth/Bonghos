package playit

import "testing"

func TestParseDaemonVersion(t *testing.T) {
	for input, want := range map[string]string{
		"playitd 1.0.10":              "1.0.10",
		"playitd version v1.2.3-rc.1": "1.2.3-rc.1",
		"playit 0.17.1\n":             "0.17.1",
	} {
		if got := parseDaemonVersion(input); got != want {
			t.Errorf("parseDaemonVersion(%q) = %q, want %q", input, got, want)
		}
	}
	if got := parseDaemonVersion("playitd development build"); got != "" {
		t.Fatalf("unexpected development version %q", got)
	}
}
