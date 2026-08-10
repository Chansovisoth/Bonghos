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

func TestDetectCommonStartupScriptNames(t *testing.T) {
	names := []string{
		"start.sh", "run.sh", "server.sh", "launch.sh", "startserver.sh", "start-server.sh", "minecraft.sh",
		"start.bat", "run.bat", "server.bat", "launch.bat", "startserver.bat", "start-server.bat",
		"start.cmd", "run.cmd",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			root := writeServer(t, map[string]string{name: "java -Xms1G -Xmx2G -jar server.jar nogui\n"})
			candidates, err := DetectStartupScripts(root, 2)
			if err != nil {
				t.Fatal(err)
			}
			if len(candidates) != 1 || candidates[0].Path != name || !candidates[0].HasJava {
				t.Fatalf("candidates=%+v, want Java startup %q", candidates, name)
			}
		})
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

func TestDetectCommonJVMConfigurationFiles(t *testing.T) {
	names := []string{
		"user_jvm_args.txt", "variables.txt", "jvm_args.txt", "jvm-args.txt", "jvm.args",
		"java_args.txt", "java-args.txt", "java.args", "args.txt", "flags.txt", ".env",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			body := "-Xms2G\n-Xmx6G\n-XX:+UseG1GC\n"
			if name == "variables.txt" || name == ".env" {
				body = "JVM_ARGS=\"-Xms2G -Xmx6G -XX:+UseG1GC\"\n"
			}
			root := writeServer(t, map[string]string{
				"start.sh": "#!/bin/bash\njava -jar server.jar nogui\n",
				name:       body,
			})
			cfg, err := DetectJVMConfig(root, "start.sh")
			if err != nil {
				t.Fatal(err)
			}
			if cfg.SourceFile != name || cfg.Xms != "2G" || cfg.Xmx != "6G" || !cfg.Editable {
				t.Fatalf("config=%+v, want editable %s with 2G/6G", cfg, name)
			}
		})
	}
}

func TestDetectAndUpdateSeparateEnvMemoryValues(t *testing.T) {
	root := writeServer(t, map[string]string{
		"start.sh": "#!/bin/bash\njava -jar server.jar nogui\n",
		".env":     "XMS=2G\nXMX=6G\nOTHER=value\n",
	})
	cfg, err := DetectJVMConfig(root, "start.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceFile != ".env" || cfg.SourceKind != "variable" || cfg.Xms != "2G" || cfg.Xmx != "6G" {
		t.Fatalf("config=%+v, want .env variable 2G/6G", cfg)
	}
	updated := UpdateJVMArgFile("XMS=2G\nXMX=6G\nOTHER=value\n", "3G", "8G")
	if updated != "XMS=3G\nXMX=8G\nOTHER=value\n" {
		t.Fatalf("updated environment file incorrectly:\n%s", updated)
	}
}

func TestDetectJVMVariableInBatchStartupScript(t *testing.T) {
	root := writeServer(t, map[string]string{
		"start.bat": "@echo off\r\nset \"JVM_ARGS=-Xms3G -Xmx7G\"\r\n%JAVA% %JVM_ARGS% -jar server.jar nogui\r\n",
	})
	cfg, err := DetectJVMConfig(root, "start.bat")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceFile != "start.bat" || cfg.SourceKind != "variable" || cfg.Xms != "3G" || cfg.Xmx != "7G" || !cfg.Editable {
		t.Fatalf("config=%+v, want editable batch JVM_ARGS 3G/7G", cfg)
	}
}

func TestForgeInternalArgumentFilesAreNeverEditable(t *testing.T) {
	internalNames := []string{"unix_args.txt", "win_args.txt", "args.txt"}
	for _, name := range internalNames {
		t.Run(name, func(t *testing.T) {
			rel := filepath.ToSlash(filepath.Join("libraries", "net", "neoforged", "loader", name))
			root := writeServer(t, map[string]string{
				"run.sh": "#!/bin/bash\njava @" + rel + " nogui\n",
				rel:      "-Xms2G\n-Xmx6G\n",
			})
			cfg, err := DetectJVMConfig(root, "run.sh")
			if err != nil {
				t.Fatal(err)
			}
			if cfg.SourceKind != "none" || cfg.Editable {
				t.Fatalf("internal %s was exposed as JVM configuration: %+v", rel, cfg)
			}
		})
	}
}

func TestRootArgsFileWinsOverReferencedForgeInternalArgs(t *testing.T) {
	root := writeServer(t, map[string]string{
		"run.sh": "#!/bin/bash\njava @libraries/net/minecraftforge/unix_args.txt @args.txt nogui\n",
		"libraries/net/minecraftforge/unix_args.txt": "-Xmx99G\n",
		"args.txt": "-Xms2G\n-Xmx6G\n",
	})
	cfg, err := DetectJVMConfig(root, "run.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceFile != "args.txt" || !cfg.Editable || cfg.Xmx != "6G" {
		t.Fatalf("config=%+v, want editable root args.txt", cfg)
	}
}

func TestParseMemoryBytes(t *testing.T) {
	for input, want := range map[string]int64{
		"512M": 512 << 20,
		"6G":   6 << 30,
		"1024": 1024,
	} {
		got, err := ParseMemoryBytes(input)
		if err != nil || got != want {
			t.Errorf("ParseMemoryBytes(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
	if _, err := ParseMemoryBytes("nope"); err == nil {
		t.Error("ParseMemoryBytes accepted invalid input")
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
