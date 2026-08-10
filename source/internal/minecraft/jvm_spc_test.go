package minecraft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A ServerPackCreator pack: variables.txt owns JAVA_ARGS and start.sh
// regenerates user_jvm_args.txt from it on every run. Editing the generated
// file looks like it worked and is reverted on the next restart — which is
// exactly what happened on a live server, where the panel recorded Xmx=5G
// while the running server went back to 4G.
func writeSPCPack(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"variables.txt": "JAVA=\"java\"\nJAVA_ARGS=\"-Xmx4G -Xms4G\"\nMINECRAFT_VERSION=1.20.1\n",
		"start.sh": `#!/usr/bin/env bash
# This script reads variables.txt. Do not edit user_jvm_args.txt directly:
# it is regenerated on every run.
source ./variables.txt
echo "${JAVA_ARGS}" > user_jvm_args.txt
${JAVA} @user_jvm_args.txt @libraries/net/neoforged/args.txt nogui
`,
		"user_jvm_args.txt": "-Xmx4G -Xms4G\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestServerPackCreatorVariablesOwnJVMSettings(t *testing.T) {
	root := writeSPCPack(t)
	cfg, err := DetectJVMConfig(root, "start.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceFile != "variables.txt" {
		t.Errorf("SourceFile = %q, want variables.txt (the file the pack reads)", cfg.SourceFile)
	}
	if cfg.SourceKind != "variable" || cfg.Variable != "JAVA_ARGS" {
		t.Errorf("SourceKind=%q Variable=%q, want variable/JAVA_ARGS", cfg.SourceKind, cfg.Variable)
	}
	if cfg.Xmx != "4G" || cfg.Xms != "4G" {
		t.Errorf("Xms=%q Xmx=%q, want 4G/4G", cfg.Xms, cfg.Xmx)
	}
	if !cfg.Editable {
		t.Error("the authoritative source should be editable")
	}
}

// A plain Forge pack, where user_jvm_args.txt really is the source, must keep
// working exactly as before.
func TestOrdinaryArgFileIsStillPreferred(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "run.sh"),
		[]byte("#!/bin/bash\njava @user_jvm_args.txt @libraries/args.txt nogui\n"), 0o755)
	os.WriteFile(filepath.Join(root, "user_jvm_args.txt"),
		[]byte("# comment\n-Xms2G\n-Xmx8G\n"), 0o644)

	cfg, err := DetectJVMConfig(root, "run.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceFile != "user_jvm_args.txt" || cfg.SourceKind != "arg_file" {
		t.Errorf("SourceFile=%q SourceKind=%q, want user_jvm_args.txt/arg_file", cfg.SourceFile, cfg.SourceKind)
	}
	if !cfg.Editable {
		t.Error("a genuine argument file should be editable")
	}
	if cfg.Xmx != "8G" {
		t.Errorf("Xmx = %q, want 8G", cfg.Xmx)
	}
}

// Some packs keep their launch flags in variables.txt without generating a
// dedicated JVM argument file. The well-known settings file is still the
// editable source even when the startup script does not use a shell `source`
// statement that the detector can follow.
func TestVariablesFileDetectedWithoutGeneratedArgFile(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "start.sh"), []byte(
		"#!/bin/bash\n. ./variables.txt\njava $JVM_ARGS -jar server.jar nogui\n"), 0o755)
	os.WriteFile(filepath.Join(root, "variables.txt"), []byte(
		"JVM_ARGS=\"-Xms2G -Xmx6G -XX:+UseG1GC\"\n"), 0o644)

	cfg, err := DetectJVMConfig(root, "start.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceFile != "variables.txt" || cfg.SourceKind != "variable" {
		t.Errorf("SourceFile=%q SourceKind=%q, want variables.txt/variable", cfg.SourceFile, cfg.SourceKind)
	}
	if cfg.Variable != "JVM_ARGS" || cfg.Xms != "2G" || cfg.Xmx != "6G" {
		t.Errorf("Variable=%q Xms=%q Xmx=%q, want JVM_ARGS/2G/6G", cfg.Variable, cfg.Xms, cfg.Xmx)
	}
	if !cfg.Editable {
		t.Error("variables.txt should be editable")
	}
}

// Saving must change variables.txt, and the value must survive the pack
// regenerating user_jvm_args.txt from it.
func TestSavingEditsVariablesAndSurvivesRegeneration(t *testing.T) {
	root := writeSPCPack(t)
	cfg, err := DetectJVMConfig(root, "start.sh")
	if err != nil {
		t.Fatal(err)
	}

	varsPath := filepath.Join(root, cfg.SourceFile)
	before, err := os.ReadFile(varsPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := UpdateJVMArgFile(string(before), "6G", "6G")
	if err := os.WriteFile(varsPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-detect: the new values must be what the panel reports.
	cfg2, err := DetectJVMConfig(root, "start.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Xmx != "6G" || cfg2.Xms != "6G" {
		t.Fatalf("after saving, Xms=%q Xmx=%q, want 6G/6G", cfg2.Xms, cfg2.Xmx)
	}

	// Simulate the pack regenerating the argument file from variables.txt on
	// the next start. The intended setting must survive.
	regenerated := "-Xmx6G -Xms6G\n"
	if err := os.WriteFile(filepath.Join(root, "user_jvm_args.txt"), []byte(regenerated), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg3, err := DetectJVMConfig(root, "start.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg3.Xmx != "6G" {
		t.Errorf("after regeneration Xmx = %q, want the intended 6G", cfg3.Xmx)
	}
}

// If the generated file is somehow the only source found, it must be reported
// as not editable rather than silently accepting writes that get reverted.
func TestGeneratedArgFileWithoutAVariableIsNotEditable(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "start.sh"), []byte(
		"#!/bin/bash\necho \"-Xmx4G\" > user_jvm_args.txt\njava @user_jvm_args.txt nogui\n"), 0o755)
	os.WriteFile(filepath.Join(root, "user_jvm_args.txt"), []byte("-Xmx4G\n"), 0o644)

	cfg, err := DetectJVMConfig(root, "start.sh")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editable {
		t.Error("a file the script rewrites at launch must not be presented as editable")
	}
	if cfg.Note == "" {
		t.Error("the reason it cannot be edited should be explained")
	}
}

// Both values on one line must both be rewritten. Applying the second
// substitution to the original line discarded the first, so editing memory in
// a JAVA_ARGS="-Xmx4G -Xms4G" style file silently kept the old Xms.
func TestUpdateJVMArgFileRewritesBothValuesOnOneLine(t *testing.T) {
	in := `JAVA="java"
JAVA_ARGS="-Xmx4G -Xms4G"
MINECRAFT_VERSION=1.20.1
`
	out := UpdateJVMArgFile(in, "6G", "8G")
	if !strings.Contains(out, "-Xms6G") {
		t.Errorf("Xms was not updated:\n%s", out)
	}
	if !strings.Contains(out, "-Xmx8G") {
		t.Errorf("Xmx was not updated:\n%s", out)
	}
	// The shell assignment must remain syntactically intact.
	if !strings.Contains(out, `JAVA_ARGS="-Xmx8G -Xms6G"`) {
		t.Errorf("the assignment was corrupted:\n%s", out)
	}
	if !strings.Contains(out, `JAVA="java"`) || !strings.Contains(out, "MINECRAFT_VERSION=1.20.1") {
		t.Errorf("unrelated lines were lost:\n%s", out)
	}
}

// A greedy value pattern used to swallow the closing quote.
func TestUpdateJVMArgFilePreservesQuoting(t *testing.T) {
	out := UpdateJVMArgFile(`JAVA_ARGS="-Xmx4G"`, "", "2G")
	if out != `JAVA_ARGS="-Xmx2G"` {
		t.Errorf("got %q, want the quoting preserved", out)
	}
}
