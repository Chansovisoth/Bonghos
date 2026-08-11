// Command bonghos is the single self-contained binary for the Bonghos
// Minecraft server management panel: control plane, Minecraft supervisor,
// console client and maintenance CLI in one executable.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/Chansovisoth/Bonghos/internal/app"
	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/database"
	"github.com/Chansovisoth/Bonghos/internal/minecraft"
	"github.com/Chansovisoth/Bonghos/internal/portability"
	"github.com/Chansovisoth/Bonghos/internal/qrcode"
	"github.com/Chansovisoth/Bonghos/internal/runtime/console"
	"github.com/Chansovisoth/Bonghos/internal/runtime/systemd"
	"github.com/Chansovisoth/Bonghos/internal/runtime/tmux"
	"github.com/Chansovisoth/Bonghos/internal/supervisor"
)

//go:embed all:webdist
var webEmbed embed.FS

var version = "0.2.0-rc.1"

func main() {
	app.Version = version

	home := flag.String("home", defaultHome(), "Bonghos home directory (BONGHOS_HOME)")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	var err error
	switch cmd {
	case "serve":
		err = cmdServe(*home)
	case "supervisor":
		err = cmdSupervisor(*home)
	case "setup":
		err = cmdSetup(*home)
	case "console":
		err = cmdConsole(*home, args)
	case "doctor":
		err = cmdDoctor(*home, args)
	case "database":
		err = cmdDatabase(*home, args)
	case "fix-permissions":
		err = cmdFixPerms(*home)
	case "export":
		err = cmdExport(*home, args)
	case "import":
		err = cmdImport(*home, args)
	case "backup":
		err = cmdBackup(*home, args)
	case "server":
		err = cmdServer(*home, args)
	case "admin":
		err = cmdAdminCreate(*home, args)
	case "service":
		err = cmdService(*home, args)
	case "user":
		err = cmdUser(*home, args)
	case "version":
		fmt.Println("bonghos", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func defaultHome() string {
	if h := os.Getenv("BONGHOS_HOME"); h != "" {
		return h
	}
	uh, err := os.UserHomeDir()
	if err != nil {
		return "./bonghos"
	}
	return filepath.Join(uh, "bonghos")
}

func usage() {
	fmt.Print(`Bonghos — Minecraft Server Management Panel

Usage: bonghos [--home DIR] <command>

Server control:
  serve                     Run the control plane (Web UI + API)     [default]
  supervisor                Run the Minecraft supervisor (used by systemd)
  server list               List server projects
  server select <slug|id>   Choose the active project
  server import <dir>       Import an existing server directory
  server start              Start the active server
  server stop               Stop gracefully (saves the world)
  server restart            Save, stop gracefully, then start again
  server force-stop [--yes] Kill immediately (recent changes may be lost)
  console [--direct]        Attach the Minecraft console (tmux, or --direct)

Accounts:
  setup                     First-run setup (Owner account, services)
  admin create              Create the first Owner account
  user list                 List accounts
  user invite [role]        Create a single-use activation link
  user disable <username>   Disable an account and revoke its sessions
  user enable <username>    Re-enable an account
  user revoke-sessions <u>  Sign an account out everywhere
  user reset-password <u>   Set a new password and revoke sessions

Backups:
  backup <type>             Create a backup (world|full|configuration)
  backup list               List backups for the active project
  backup verify <id>        Re-verify an archive and its checksum
  backup restore <id>       Restore (requires a stopped server)

Maintenance:
  doctor [--repair]         Diagnose (and optionally repair) the installation
  database checkpoint       Integrity-check and checkpoint the SQLite database
  fix-permissions           Restore expected file modes inside the home
  export [flags]            Create a portable export archive
  import [--force] <file>   Import a portable export archive
  service <verb>            install | repair | uninstall | status
  version                   Print the version

Environment:
  BONGHOS_HOME      Overrides the default home (~/bonghos)
`)
}

// ---------------------------------------------------------------------------
// serve
// ---------------------------------------------------------------------------

func webFS() fs.FS {
	sub, err := fs.Sub(webEmbed, "webdist")
	if err != nil {
		return nil
	}
	if f, err := sub.Open("index.html"); err == nil {
		f.Close()
		return sub
	}
	return nil
}

func cmdServe(home string) error {
	a, err := app.New(home, webFS())
	if err != nil {
		return err
	}
	defer a.Close()

	// Refuse to serve before an Owner exists (unless setup was completed).
	var owners int
	_ = a.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role=? AND disabled=0`, authorization.RoleOwner).Scan(&owners)
	if owners == 0 {
		fmt.Println("No Owner account exists yet. Run: bonghos setup")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return a.Serve(ctx)
}

// ---------------------------------------------------------------------------
// supervisor (invoked by systemd; runs one Minecraft server)
// ---------------------------------------------------------------------------

func cmdSupervisor(home string) error {
	if err := config.InitHome(home); err != nil {
		return err
	}
	cfg, err := config.Load(home)
	if err != nil {
		return err
	}
	db, err := database.Open(filepath.Join(home, config.FileDatabase))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return err
	}

	var instID int64
	if err := db.QueryRow(`SELECT instance_id FROM active_instance WHERE id=1`).Scan(&instID); err != nil || instID == 0 {
		return fmt.Errorf("no active server project selected")
	}
	var slug, relDir, script, javaSel, restartPolicy string
	var external, restartDelay int
	err = db.QueryRow(`SELECT slug, server_directory, external_directory, startup_script,
		java_selection, restart_policy, restart_delay_seconds
		FROM instances WHERE id=?`, instID).
		Scan(&slug, &relDir, &external, &script, &javaSel, &restartPolicy, &restartDelay)
	if err != nil {
		return fmt.Errorf("loading active project: %w", err)
	}
	dir := relDir
	if external == 0 {
		dir = filepath.Join(home, relDir)
	}
	javaPath, err := minecraft.ResolveJava(javaSel)
	if err != nil {
		return fmt.Errorf("java selection: %w", err)
	}

	sup := supervisor.New(supervisor.Config{
		Home:                home,
		InstanceID:          instID,
		ServerDir:           dir,
		StartupScript:       script,
		JavaPath:            javaPath,
		RestartPolicy:       restartPolicy,
		RestartDelaySeconds: restartDelay,
		GracefulStopSeconds: cfg.GracefulStopSeconds,
	})
	srv := &console.Server{Home: home, Sup: sup}
	if err := srv.Start(); err != nil {
		return fmt.Errorf("console socket: %w", err)
	}
	defer srv.Close()

	_ = slug // tmux console attaches on demand via `bonghos console`
	_ = tmux.Installed

	ctx, cancel := context.WithCancel(context.Background())
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		_ = sup.Stop(true)
		// Give the graceful stop room; supervisor.Run returns when stopped.
		go func() {
			<-sig
			_ = sup.ForceStop()
			cancel()
		}()
	}()

	return sup.Run(ctx)
}

// ---------------------------------------------------------------------------
// setup
// ---------------------------------------------------------------------------

func cmdSetup(home string) error {
	fmt.Println("Bonghos first-run setup")
	fmt.Println("Home:", home)
	a, err := app.New(home, nil)
	if err != nil {
		return err
	}
	defer a.Close()

	var owners int
	_ = a.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role=? AND disabled=0`, authorization.RoleOwner).Scan(&owners)
	if owners > 0 {
		fmt.Println("An Owner account already exists; setup will not create another.")
	} else {
		if err := setupOwner(a); err != nil {
			return err
		}
	}

	// systemd services
	if systemd.Available() {
		fmt.Print("Install systemd user services (recommended)? [Y/n]: ")
		if readLine() != "n" {
			if err := systemd.Install(home, a.Cfg.GracefulStopSeconds); err != nil {
				fmt.Println("Service install failed:", err)
			} else {
				fmt.Println("Services installed. Enable bonghos.service when you are ready to start the panel.")
				if hint, err := systemd.LingerHint(); err == nil && hint != "" {
					fmt.Println(hint)
				}
			}
		}
	} else {
		fmt.Println("Note: systemd user services are unavailable; Bonghos will run in the foreground.")
	}

	fmt.Printf("\nSetup complete. Start the panel with:\n  systemctl --user start %s\nor:\n  bonghos serve\n\nThen open http://%s:%d\n",
		systemd.ServiceControlPlane, a.Cfg.BindAddress, a.Cfg.Port)
	return nil
}

func setupOwner(a *app.App) error {
	fmt.Println("\nCreate the Owner account.")
	var username string
	for {
		fmt.Print("Username: ")
		username = readLine()
		if err := auth.ValidateUsername(username); err != nil {
			fmt.Println(" ", err)
			continue
		}
		break
	}
	var password string
	for {
		fmt.Print("Password (min 10 chars): ")
		p1, err := readSecret()
		if err != nil {
			return err
		}
		if err := auth.ValidatePassword(p1); err != nil {
			fmt.Println(" ", err)
			continue
		}
		fmt.Print("Confirm password: ")
		p2, err := readSecret()
		if err != nil {
			return err
		}
		if p1 != p2 {
			fmt.Println("  Passwords do not match.")
			continue
		}
		password = p1
		break
	}
	secret, err := auth.GenerateTOTPSecret()
	if err != nil {
		return err
	}
	fmt.Println("\nTwo-factor authentication (TOTP) is mandatory.")
	printTOTPEnrolment(username, secret)
	for {
		fmt.Print("Enter the 6-digit code from your app to confirm: ")
		code := readLine()
		if auth.VerifyTOTP(secret, code, time.Now()) {
			break
		}
		fmt.Println("  Code did not match; check your device clock and try again.")
	}
	user, recovery, err := a.Auth.CreateUser(username, password, secret, authorization.RoleOwner)
	if err != nil {
		return err
	}
	fmt.Println("\nOwner account created:", user.Username)
	fmt.Println("\nRECOVERY CODES — store these safely; each works once if you lose your authenticator:")
	for _, c := range recovery {
		fmt.Println("  ", c)
	}
	fmt.Println()
	return nil
}

func readLine() string {
	var s string
	fmt.Scanln(&s)
	return strings.TrimSpace(s)
}

// printTOTPEnrolment shows the enrolment QR code when the terminal can display
// one, always followed by the secret and URI. Rendering is best-effort: any
// failure falls through to the manual values rather than interrupting setup.
func printTOTPEnrolment(username, secret string) {
	uri := auth.TOTPProvisioningURI(username, secret)

	if art, err := qrcode.Stdout(uri); err == nil {
		fmt.Println("\nScan this QR code with your authenticator app:")
		fmt.Println()
		fmt.Print(art)
		fmt.Println("\nIf scanning does not work, enter this secret manually:")
	} else {
		// Not a terminal, too narrow, or encoding failed. The manual path is
		// the authoritative one anyway.
		fmt.Println("Add this secret to your authenticator app:")
	}

	fmt.Println("\n  Secret:", secret)
	fmt.Println("  URI:   ", uri)
	fmt.Println()
}

func readSecret() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		return string(b), err
	}
	return readLine(), nil
}

// ---------------------------------------------------------------------------
// console
// ---------------------------------------------------------------------------

func cmdConsole(home string, args []string) error {
	direct := false
	for _, arg := range args {
		switch arg {
		case "--direct":
			direct = true
		case "attach":
			// Internal tmux child invocation: bonghos console attach --direct.
		default:
			return fmt.Errorf("usage: bonghos console [--direct]")
		}
	}
	if !direct {
		self, err := os.Executable()
		if err != nil {
			return err
		}
		return tmux.Console(self, home)
	}
	return cmdConsoleDirect(home)
}

func cmdConsoleDirect(home string) error {
	c, err := console.Dial(home)
	if err != nil {
		return fmt.Errorf("cannot connect to the supervisor (is the server running?): %w", err)
	}
	defer c.Close()
	fmt.Println("Connected. Type Minecraft commands; Ctrl+C to detach (the server keeps running).")
	return c.Interactive(os.Stdin, os.Stdout)
}

// ---------------------------------------------------------------------------
// doctor / fix-permissions
// ---------------------------------------------------------------------------

func cmdDatabase(home string, args []string) error {
	if len(args) != 1 || args[0] != "checkpoint" {
		return errors.New("usage: bonghos database checkpoint")
	}
	db, err := database.OpenForMaintenance(filepath.Join(home, config.FileDatabase))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.IntegrityCheck(db); err != nil {
		return err
	}
	if err := database.Checkpoint(db); err != nil {
		return err
	}
	fmt.Println("Database integrity check passed and WAL checkpoint completed.")
	return nil
}

func cmdDoctor(home string, args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	repair := fs.Bool("repair", false, "apply safe automatic repairs")
	fixPerms := fs.Bool("fix-permissions", false, "restore expected file modes inside the home")
	jsonOut := fs.Bool("json", false, "machine-readable output")
	_ = fs.Parse(args)

	// `bonghos doctor --fix-permissions` is equivalent to the standalone
	// `bonghos fix-permissions` command; both are documented.
	if *fixPerms {
		return cmdFixPerms(home)
	}

	rep, _ := portability.Doctor(home, *repair)
	if *jsonOut {
		b, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(b))
	} else {
		for _, c := range rep.Checks {
			mark := map[portability.CheckStatus]string{
				portability.StatusOK: "✓", portability.StatusWarning: "!",
				portability.StatusError: "✗", portability.StatusFixed: "✔ fixed",
				portability.StatusSkipped: "-",
			}[c.Status]
			line := fmt.Sprintf("[%s] %s", mark, c.Name)
			if c.Detail != "" {
				line += " — " + c.Detail
			}
			fmt.Println(line)
		}
		fmt.Printf("\n%d error(s), %d warning(s), %d fixed\n", rep.Errors, rep.Warnings, rep.Fixed)
	}
	if rep.Errors > 0 {
		os.Exit(1)
	}
	return nil
}

func cmdFixPerms(home string) error {
	rep, err := portability.FixPermissions(home)
	if err != nil {
		return err
	}
	for _, c := range rep.Checks {
		fmt.Printf("[%s] %s %s\n", c.Status, c.Name, c.Detail)
	}
	return nil
}

// ---------------------------------------------------------------------------
// export / import
// ---------------------------------------------------------------------------

func cmdExport(home string, args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	scope := fs.String("scope", "complete", "complete | configuration_only | system_data | servers | backups")
	secrets := fs.Bool("include-secrets", false, "include secret.key and the account database")
	output := fs.String("output", "", "output archive path (.tar.zst)")
	_ = fs.Parse(args)

	if *secrets {
		fmt.Println("WARNING: the export will contain your secret key and account database.")
		fmt.Println("Anyone holding this file can decrypt stored TOTP secrets. Protect it accordingly.")
		fmt.Print("Type 'yes' to continue: ")
		if readLine() != "yes" {
			return fmt.Errorf("export cancelled")
		}
	} else {
		fmt.Println("Note: without --include-secrets the export excludes secret.key and the account")
		fmt.Println("database; after import you will re-run setup to create accounts.")
	}
	out, err := portability.Export(home, portability.ExportOptions{
		Scope:          portability.ExportScope(*scope),
		IncludeSecrets: *secrets,
		Output:         *output,
		Version:        version,
	})
	if err != nil {
		return err
	}
	fmt.Println("Export written and verified:", out)
	return nil
}

func cmdImport(home string, args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	force := fs.Bool("force", false, "merge over an existing installation (safety copies are made)")
	_ = fs.Parse(args)
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: bonghos import [--force] <archive.tar.zst>")
	}
	m, err := portability.Import(fs.Arg(0), home, *force)
	if err != nil {
		return err
	}
	fmt.Printf("Imported Bonghos export (created %s from %s/%s).\n", m.CreatedAt, m.SourceOS, m.SourceArchitecture)
	if !m.IncludesSecrets {
		fmt.Println("This export did not include secrets: run 'bonghos setup' to create the Owner account.")
	}
	fmt.Println("Next: bonghos doctor --repair && bonghos service repair (if using systemd)")
	return nil
}

// ---------------------------------------------------------------------------
// backup (CLI)
// ---------------------------------------------------------------------------

func cmdBackup(home string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: bonghos backup <create world|full|configuration | list | verify ID | restore ID>")
	}
	// Sub-verbs that operate on existing backups.
	switch args[0] {
	case "list", "verify", "restore":
		a, err := app.New(home, nil)
		if err != nil {
			return err
		}
		defer a.Close()
		switch args[0] {
		case "list":
			return backupList(a)
		case "verify":
			return backupVerify(a, args[1:])
		default:
			return backupRestore(a, args[1:])
		}
	}
	// "create" is optional: `bonghos backup full` and `bonghos backup create full`.
	if args[0] == "create" {
		args = args[1:]
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: bonghos backup <world|full|configuration>")
	}
	var t backup.Type
	switch args[0] {
	case "world":
		t = backup.TypeWorld
	case "full":
		t = backup.TypeFull
	case "configuration":
		t = backup.TypeConfig
	default:
		return fmt.Errorf("type must be world, full or configuration")
	}
	a, err := app.New(home, nil)
	if err != nil {
		return err
	}
	defer a.Close()
	inst, err := a.ActiveInstance()
	if err != nil {
		return err
	}
	mode := "offline"
	if a.Runner.Online() {
		mode = "online"
	}
	fmt.Printf("Creating %s backup of %s (%s mode)…\n", args[0], inst.Slug, mode)
	rec, err := a.RunBackup(context.Background(), inst, t, mode, "manual", 0)
	if err != nil {
		return err
	}
	fmt.Printf("Backup %s created (%.1f MiB, %d files, verified).\n",
		rec.BackupID, float64(rec.CompressedSize)/(1<<20), rec.FileCount)
	return nil
}

// ---------------------------------------------------------------------------
// service
// ---------------------------------------------------------------------------

func cmdService(home string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: bonghos service <install|repair|uninstall|status>")
	}
	cfg, err := config.Load(home)
	if err != nil {
		return err
	}
	switch args[0] {
	case "install":
		if err := systemd.Install(home, cfg.GracefulStopSeconds); err != nil {
			return err
		}
		fmt.Println("Installed and enabled", systemd.ServiceControlPlane, "and", systemd.ServiceMinecraft)
		if hint, err := systemd.LingerHint(); err == nil && hint != "" {
			fmt.Println(hint)
		}
	case "repair":
		if err := systemd.Repair(home, cfg.GracefulStopSeconds); err != nil {
			return err
		}
		fmt.Println("Service unit files regenerated for home", home)
	case "uninstall":
		if err := systemd.Uninstall(); err != nil {
			return err
		}
		fmt.Println("Services removed. Bonghos data was left untouched.")
	case "status":
		fmt.Println(systemd.ServiceControlPlane+":", systemd.Status(systemd.ServiceControlPlane))
		fmt.Println(systemd.ServiceMinecraft+":", systemd.Status(systemd.ServiceMinecraft))
	default:
		return fmt.Errorf("unknown service verb %q", args[0])
	}
	return nil
}

// ---------------------------------------------------------------------------
// user
// ---------------------------------------------------------------------------

func cmdUser(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: bonghos user <list|invite|disable|enable|revoke-sessions|reset-password>")
	}
	a, err := app.New(home, nil)
	if err != nil {
		return err
	}
	defer a.Close()
	switch args[0] {
	case "list":
		users, err := a.Auth.ListUsers()
		if err != nil {
			return err
		}
		for _, u := range users {
			state := "active"
			if u.Disabled {
				state = "disabled"
			}
			fmt.Printf("%-24s %-8s %s\n", u.Username, u.Role, state)
		}
	case "reset-password":
		if len(args) != 2 {
			return fmt.Errorf("usage: bonghos user reset-password USERNAME")
		}
		u, err := a.Auth.UserByName(args[1])
		if err != nil {
			return fmt.Errorf("no such user")
		}
		fmt.Print("New password: ")
		p, err := readSecret()
		if err != nil {
			return err
		}
		if err := auth.ValidatePassword(p); err != nil {
			return err
		}
		if err := a.Auth.ResetPassword(u.ID, p); err != nil {
			return err
		}
		_ = a.Auth.RevokeAllSessions(u.ID)
		fmt.Println("Password updated; all sessions revoked.")
	case "invite":
		return userInvite(a, args[1:])
	case "disable":
		return userSetDisabled(a, args[1:], true)
	case "enable":
		return userSetDisabled(a, args[1:], false)
	case "revoke-sessions":
		return userRevokeSessions(a, args[1:])
	default:
		return fmt.Errorf("unknown user verb %q", args[0])
	}
	return nil
}
