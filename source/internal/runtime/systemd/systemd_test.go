package systemd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedUnitsQuotePathsAndApplyPortableHardening(t *testing.T) {
	home := `/srv/Bonghos Server/100% ready`
	bin := home + `/system/bin/bonghos`
	control := ControlPlaneUnit(home, bin)
	minecraft := MinecraftUnit(home, bin, 90)

	for _, unit := range []string{control, minecraft} {
		for _, want := range []string{
			`Environment="BONGHOS_HOME=/srv/Bonghos Server/100%% ready"`,
			`WorkingDirectory=/srv/Bonghos\x20Server/100%%\x20ready`,
			`ExecStart="/srv/Bonghos Server/100%% ready/system/bin/bonghos" --home "/srv/Bonghos Server/100%% ready"`,
			`NoNewPrivileges=yes`,
			`RestrictSUIDSGID=yes`,
		} {
			if !strings.Contains(unit, want) {
				t.Errorf("generated unit does not contain %q:\n%s", want, unit)
			}
		}
	}

	if !strings.Contains(control, `PrivateTmp=yes`) {
		t.Error("control-plane unit does not contain PrivateTmp=yes")
	}
	if !strings.Contains(control, "Restart=always\nRestartSec=5s") {
		t.Error("control-plane unit must restart after unexpected clean or failed exits")
	}
	if !strings.Contains(minecraft, "Restart=on-failure\nRestartSec=5s") {
		t.Error("Minecraft unit must not restart after an intentional clean stop")
	}
	// ProtectControlGroups, ProtectKernelModules and ProtectKernelTunables
	// implicitly alter the capability bounding set. Some restricted Linux
	// hosts reject those changes in user units with 218/CAPABILITIES before
	// Bonghos can start. The process is already an unprivileged user service.
	for _, incompatible := range []string{
		`ProtectControlGroups=`,
		`ProtectKernelModules=`,
		`ProtectKernelTunables=`,
	} {
		if strings.Contains(control, incompatible) {
			t.Errorf("control-plane unit contains non-portable hardening %q", incompatible)
		}
	}
	if strings.Contains(minecraft, "\n[Install]\n") {
		t.Error("Minecraft unit must not be installable independently")
	}
}

func TestGeneratedUnitsPassSystemdAnalyze(t *testing.T) {
	analyze, err := exec.LookPath("systemd-analyze")
	if err != nil {
		t.Skip("systemd-analyze is not installed")
	}

	root := filepath.Join(t.TempDir(), "Bonghos Server")
	bin := filepath.Join(root, "system", "bin", "bonghos")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	controlPath := filepath.Join(t.TempDir(), ServiceControlPlane)
	minecraftPath := filepath.Join(t.TempDir(), ServiceMinecraft)
	if err := os.WriteFile(controlPath, []byte(ControlPlaneUnit(root, bin)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(minecraftPath, []byte(MinecraftUnit(root, bin, 90)), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(analyze, "verify", controlPath, minecraftPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("systemd-analyze verify: %v\n%s", err, out)
	}
}
