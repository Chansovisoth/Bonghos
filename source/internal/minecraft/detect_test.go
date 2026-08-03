package minecraft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeServer(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestDetectStartupScripts(t *testing.T) {
	root := writeServer(t, map[string]string{
		"startserver.sh": "#!/bin/bash\njava @user_jvm_args.txt @libraries/net/neoforged/args.txt nogui\n",
		"random.sh":      "#!/bin/bash\necho not a server\n",
	})
	cands, err := DetectStartupScripts(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) == 0 {
		t.Fatal("no candidates found")
	}
	if cands[0].Path != "startserver.sh" {
		t.Errorf("top candidate = %q, want startserver.sh (got %+v)", cands[0].Path, cands)
	}
	if cands[0].Modloader != "neoforge" {
		t.Errorf("modloader = %q, want neoforge", cands[0].Modloader)
	}
}

func TestDetectStartupScriptUsesServerPackVariablesForModloader(t *testing.T) {
	root := writeServer(t, map[string]string{
		"start.sh":      "echo supports Forge and NeoForge\njava @user_jvm_args.txt @libraries/net/minecraftforge/forge/1.20.1-47.4.0/unix_args.txt nogui\n",
		"variables.txt": "# generated server pack\nMODLOADER=Forge\nMODLOADER_VERSION=47.4.0\n",
	})
	cands, err := DetectStartupScripts(root, 2)
	if err != nil || len(cands) == 0 {
		t.Fatalf("cands=%v err=%v", cands, err)
	}
	if cands[0].Modloader != "forge" {
		t.Fatalf("modloader = %q, want forge", cands[0].Modloader)
	}
}

func TestDetectStartupScriptIgnoresNeoForgeInComments(t *testing.T) {
	root := writeServer(t, map[string]string{
		"run.sh": "# Forge and NeoForge are both supported by this template\njava @user_jvm_args.txt @libraries/net/minecraftforge/forge/1.20.1-47.4.0/unix_args.txt nogui\n",
	})
	cands, err := DetectStartupScripts(root, 2)
	if err != nil || len(cands) == 0 {
		t.Fatalf("cands=%v err=%v", cands, err)
	}
	if cands[0].Modloader != "forge" {
		t.Fatalf("modloader = %q, want forge", cands[0].Modloader)
	}
}

func TestDetectInteractivePrompt(t *testing.T) {
	root := writeServer(t, map[string]string{
		"start.sh": "#!/bin/bash\njava -jar server.jar\nread -n 1 -p \"Press any key\"\n",
	})
	cands, err := DetectStartupScripts(root, 2)
	if err != nil || len(cands) == 0 {
		t.Fatalf("cands=%v err=%v", cands, err)
	}
	if len(cands[0].Interactive) == 0 {
		t.Error("interactive prompt not flagged")
	}
}

func TestDetectJVMConfigArgFile(t *testing.T) {
	root := writeServer(t, map[string]string{
		"run.sh":            "#!/bin/bash\njava @user_jvm_args.txt @libraries/args.txt \"$@\"\n",
		"user_jvm_args.txt": "# comment\n-Xms2G\n-Xmx8G\n-XX:+UseG1GC\n",
	})
	cfg, err := DetectJVMConfig(root, "run.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Xms != "2G" || cfg.Xmx != "8G" {
		t.Errorf("Xms=%q Xmx=%q", cfg.Xms, cfg.Xmx)
	}
	if !cfg.Editable {
		t.Error("dedicated arg file should be editable")
	}
	if cfg.SourceFile != "user_jvm_args.txt" {
		t.Errorf("SourceFile=%q", cfg.SourceFile)
	}
}

func TestDetectJVMConfigVariable(t *testing.T) {
	root := writeServer(t, map[string]string{
		"start.sh": "#!/bin/bash\nJAVA_ARGS=\"-Xms1G -Xmx6G\"\njava $JAVA_ARGS -jar forge.jar nogui\n",
	})
	cfg, err := DetectJVMConfig(root, "start.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Xmx != "6G" {
		t.Errorf("variable Xmx=%q want 6G", cfg.Xmx)
	}
}

func TestUpdateJVMArgFile(t *testing.T) {
	in := "# keep this comment\n-Xms2G\n-Xmx8G\n-XX:+UseG1GC\n"
	out := UpdateJVMArgFile(in, "4G", "12G")
	if !strings.Contains(out, "-Xms4G") || !strings.Contains(out, "-Xmx12G") {
		t.Errorf("memory not updated: %q", out)
	}
	if !strings.Contains(out, "# keep this comment") || !strings.Contains(out, "-XX:+UseG1GC") {
		t.Errorf("unrelated content lost: %q", out)
	}
}

func TestValidateMemory(t *testing.T) {
	host := int64(16) << 30
	if err := ValidateMemory("2G", "8G", host); err != nil {
		t.Errorf("valid memory rejected: %v", err)
	}
	if err := ValidateMemory("8G", "2G", host); err == nil {
		t.Error("Xms > Xmx accepted")
	}
	if err := ValidateMemory("2G", "64G", host); err == nil {
		t.Error("Xmx over host RAM accepted")
	}
	if err := ValidateMemory("2G", "banana", host); err == nil {
		t.Error("garbage Xmx accepted")
	}
}

func TestParseLogLine(t *testing.T) {
	join := `[12:34:56] [Server thread/INFO]: Steve joined the game`
	leave := `[12:35:56] [Server thread/INFO]: Steve left the game`
	if ev := ParseLogLine(join); ev == nil || ev.Kind != "joined" || ev.Player != "Steve" {
		t.Errorf("join parse: %+v", ev)
	}
	if ev := ParseLogLine(leave); ev == nil || ev.Kind != "left" || ev.Player != "Steve" {
		t.Errorf("leave parse: %+v", ev)
	}
	if ev := ParseLogLine("garbage line that matches nothing"); ev != nil {
		t.Errorf("noise produced event: %+v", ev)
	}
}
