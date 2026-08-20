package playit

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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

func TestVersionProbesPreferPackagedCLISubcommand(t *testing.T) {
	home := t.TempDir()
	cli := filepath.Join(home, "system", "bin", "playit-cli")
	if err := os.MkdirAll(filepath.Dir(cli), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cli, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	probes := versionProbes(home)
	if len(probes) == 0 {
		t.Fatal("expected a version probe")
	}
	if probes[0].executable != cli || !reflect.DeepEqual(probes[0].args, []string{"version"}) {
		t.Fatalf("first probe = %#v, want %q version", probes[0], cli)
	}
}
