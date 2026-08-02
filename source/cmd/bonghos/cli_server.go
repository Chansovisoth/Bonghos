package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/app"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
	"github.com/Chansovisoth/Bonghos/internal/backup"
	"github.com/Chansovisoth/Bonghos/internal/instance"
)

// ---------------------------------------------------------------------------
// bonghos server <list|import|select|start|stop|restart|force-stop>
// ---------------------------------------------------------------------------

func cmdServer(home string, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: bonghos server <list|import|select|start|stop|restart|force-stop>")
	}
	a, err := app.New(home, nil)
	if err != nil {
		return err
	}
	defer a.Close()

	verb, rest := args[0], args[1:]
	switch verb {
	case "list":
		return serverList(a)
	case "select":
		return serverSelect(a, rest)
	case "import":
		return serverImport(a, rest)
	case "start":
		return serverStart(a)
	case "stop":
		return serverStop(a)
	case "restart":
		return serverRestart(a)
	case "force-stop":
		return serverForceStop(a, rest)
	default:
		return fmt.Errorf("unknown server verb %q", verb)
	}
}

func serverList(a *app.App) error {
	all, err := a.Instances.List()
	if err != nil {
		return err
	}
	if len(all) == 0 {
		fmt.Println("No server projects yet. Add one in the Web UI, or:")
		fmt.Println("  bonghos server import <directory>")
		return nil
	}
	activeID, _ := a.Instances.ActiveID()
	state, _ := a.Runner.State()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSLUG\tDISPLAY NAME\tSOURCE\tACTIVE\tSTATE")
	for _, i := range all {
		active, st := "", "-"
		if i.ID == activeID {
			active, st = "*", state
		}
		name := i.DisplayName
		if i.ExternalDirectory {
			name += " (external)"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n", i.ID, i.Slug, name, i.SourceType, active, st)
	}
	return w.Flush()
}

// resolveInstance accepts a numeric ID or a slug.
func resolveInstance(a *app.App, ref string) (*instance.Instance, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return a.Instances.ByID(id)
	}
	return a.Instances.BySlug("minecraft-java-modded", ref)
}

func serverSelect(a *app.App, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: bonghos server select <slug|id>")
	}
	if a.Runner.Online() {
		return errors.New("a server is currently running; stop it before switching the active project")
	}
	inst, err := resolveInstance(a, args[0])
	if err != nil {
		return fmt.Errorf("no such server project: %s", args[0])
	}
	if err := a.Instances.SetActive(inst.ID); err != nil {
		return err
	}
	fmt.Printf("Active project is now %s (%s).\n", inst.DisplayName, inst.Slug)
	return nil
}

func serverImport(a *app.App, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: bonghos server import <directory> [display name]")
	}
	dir := args[0]
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return fmt.Errorf("not a readable directory: %s", dir)
	}
	name := strings.TrimSpace(strings.Join(args[1:], " "))
	if name == "" {
		fmt.Print("Project display name: ")
		name = readLine()
	}
	if name == "" {
		return errors.New("a project display name is required")
	}
	slug := instance.GenerateSlug(name)
	fmt.Printf("Folder slug: %s\n", slug)
	if err := instance.ValidateSlug(slug); err != nil {
		return err
	}
	fmt.Printf("Copy %s into Bonghos as %q? [y/N]: ", dir, name)
	if !strings.HasPrefix(strings.ToLower(readLine()), "y") {
		fmt.Println("Cancelled.")
		return nil
	}
	inst, err := a.ImportDirectoryCLI(context.Background(), name, slug, dir)
	if err != nil {
		return err
	}
	fmt.Printf("Imported %s into %s\n", inst.DisplayName, inst.AbsoluteDir(a.Home))
	if inst.StartupScript != "" {
		fmt.Println("Detected startup script:", inst.StartupScript)
	} else {
		fmt.Println("No startup script detected — select one in the Web UI before starting.")
	}
	return nil
}

func requireActive(a *app.App) (*instance.Instance, error) {
	inst, err := a.ActiveInstance()
	if err != nil {
		return nil, errors.New("no active server project selected (try: bonghos server select <slug>)")
	}
	return inst, nil
}

func serverStart(a *app.App) error {
	inst, err := requireActive(a)
	if err != nil {
		return err
	}
	if a.Runner.Online() {
		return errors.New("the server is already running")
	}
	fmt.Printf("Starting %s …\n", inst.DisplayName)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := a.Runner.Start(ctx); err != nil {
		return err
	}
	fmt.Println("Started. Watch the console with: bonghos console")
	return nil
}

func serverStop(a *app.App) error {
	if !a.Runner.Online() {
		return errors.New("the server is not running")
	}
	fmt.Println("Stopping gracefully (saving the world; this can take a while) …")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := a.Runner.Stop(ctx); err != nil {
		return err
	}
	fmt.Println("Stopped.")
	return nil
}

func serverRestart(a *app.App) error {
	if _, err := requireActive(a); err != nil {
		return err
	}
	fmt.Println("Restarting (save, graceful stop, wait for exit, start) …")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if err := a.Runner.Restart(ctx); err != nil {
		return err
	}
	fmt.Println("Restarted.")
	return nil
}

func serverForceStop(a *app.App, args []string) error {
	if !a.Runner.Online() {
		return errors.New("the server is not running")
	}
	confirmed := false
	for _, s := range args {
		if s == "--yes" || s == "-y" {
			confirmed = true
		}
	}
	if !confirmed {
		fmt.Println("Force stop kills the server process group immediately.")
		fmt.Println("RECENT WORLD CHANGES MAY BE LOST. Use 'bonghos server stop' unless that already failed.")
		fmt.Print("Type FORCE to continue: ")
		if readLine() != "FORCE" {
			fmt.Println("Cancelled.")
			return nil
		}
	}
	if err := a.Runner.ForceStop(); err != nil {
		return err
	}
	fmt.Println("Force stopped. This was recorded in the audit log.")
	return nil
}

// ---------------------------------------------------------------------------
// bonghos admin create   (first Owner, equivalent to the setup account step)
// ---------------------------------------------------------------------------

func cmdAdminCreate(home string, args []string) error {
	if len(args) != 1 || args[0] != "create" {
		return errors.New("usage: bonghos admin create")
	}
	a, err := app.New(home, nil)
	if err != nil {
		return err
	}
	defer a.Close()

	var owners int
	_ = a.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role='owner' AND disabled=0`).Scan(&owners)
	if owners > 0 {
		fmt.Println("An Owner account already exists.")
		fmt.Println("Additional accounts are created by invitation: bonghos user invite")
		return nil
	}
	return setupOwner(a)
}

// ---------------------------------------------------------------------------
// bonghos user invite|disable|enable|revoke-sessions  (list/reset-password
// live in main.go)
// ---------------------------------------------------------------------------

func userInvite(a *app.App, args []string) error {
	role := authorization.RoleMember
	if len(args) >= 1 {
		r := authorization.Role(strings.ToLower(args[0]))
		if !authorization.ValidRole(r) {
			return errors.New("role must be admin, member or viewer")
		}
		role = r
	} else {
		fmt.Print("Role [admin/member/viewer] (member): ")
		in := strings.ToLower(strings.TrimSpace(readLine()))
		if in != "" {
			r := authorization.Role(in)
			if !authorization.ValidRole(r) {
				return errors.New("role must be admin, member or viewer")
			}
			role = r
		}
	}
	if role == authorization.RoleOwner {
		return errors.New("invitations cannot grant the Owner role")
	}
	// Invitations are attributed to an Owner; the CLI acts on behalf of the
	// first active Owner, since there is no web session here.
	var ownerID int64
	if err := a.DB.QueryRow(
		`SELECT id FROM users WHERE role='owner' AND disabled=0 ORDER BY id LIMIT 1`,
	).Scan(&ownerID); err != nil {
		return errors.New("no active Owner account exists yet (run: bonghos setup)")
	}
	inv, err := a.Auth.CreateInvitation(ownerID, role, 48*time.Hour)
	if err != nil {
		return err
	}
	fmt.Printf("\nInvitation created for role %s.\n", role)
	fmt.Printf("Expires: %s\n\n", inv.ExpiresAt.Format(time.RFC1123))
	fmt.Println("Send this single-use activation link to the person:")
	fmt.Printf("\n  http://%s:%d/activate/%s\n\n", a.Cfg.BindAddress, a.Cfg.Port, inv.Token)
	fmt.Println("They choose their own username, password and authenticator secret.")
	fmt.Println("The link works once and cannot be shown again.")
	return nil
}

func userSetDisabled(a *app.App, args []string, disabled bool) error {
	verb := "disable"
	if !disabled {
		verb = "enable"
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: bonghos user %s USERNAME", verb)
	}
	u, err := a.Auth.UserByName(args[0])
	if err != nil {
		return errors.New("no such user")
	}
	if disabled && u.Role == authorization.RoleOwner {
		var owners int
		_ = a.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role='owner' AND disabled=0`).Scan(&owners)
		if owners <= 1 {
			return errors.New("refusing to disable the last active Owner")
		}
	}
	if err := a.Auth.SetDisabled(u.ID, disabled); err != nil {
		return err
	}
	if disabled {
		_ = a.Auth.RevokeAllSessions(u.ID)
		fmt.Printf("%s disabled; all their sessions were revoked.\n", u.Username)
	} else {
		fmt.Printf("%s enabled.\n", u.Username)
	}
	return nil
}

func userRevokeSessions(a *app.App, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: bonghos user revoke-sessions USERNAME")
	}
	u, err := a.Auth.UserByName(args[0])
	if err != nil {
		return errors.New("no such user")
	}
	if err := a.Auth.RevokeAllSessions(u.ID); err != nil {
		return err
	}
	fmt.Printf("All sessions for %s revoked; they must sign in again.\n", u.Username)
	return nil
}

// ---------------------------------------------------------------------------
// bonghos backup list|verify|restore   (create lives in main.go)
// ---------------------------------------------------------------------------

func backupList(a *app.App) error {
	inst, err := requireActive(a)
	if err != nil {
		return err
	}
	recs, err := a.Backups.List(inst.ID)
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		fmt.Printf("No backups yet for %s. Create one with: bonghos backup full\n", inst.Slug)
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BACKUP ID\tCREATED\tTYPE\tMODE\tSIZE\tFILES\tVERIFIED\tPROTECTED")
	for _, r := range recs {
		prot := ""
		if r.Protected {
			prot = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.1f MiB\t%d\t%s\t%s\n",
			r.BackupID, r.CreatedAt, r.BackupType, r.ConsistencyMode,
			float64(r.CompressedSize)/(1<<20), r.FileCount, r.VerificationStatus, prot)
	}
	return w.Flush()
}

func backupVerify(a *app.App, args []string) error {
	if len(args) != 1 {
		return errors.New("usage: bonghos backup verify <backup-id>")
	}
	fmt.Println("Verifying archive contents and checksum …")
	if err := a.Backups.Verify(args[0]); err != nil {
		return fmt.Errorf("verification FAILED: %w", err)
	}
	fmt.Println("Backup verified.")
	return nil
}

func backupRestore(a *app.App, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: bonghos backup restore <backup-id> [--scope full_server|world_only|configuration_only]")
	}
	id := args[0]
	scope := "full_server"
	for i := 1; i < len(args); i++ {
		if args[i] == "--scope" && i+1 < len(args) {
			scope = args[i+1]
		}
	}
	switch scope {
	case "full_server", "world_only", "configuration_only":
	default:
		return errors.New("scope must be full_server, world_only or configuration_only")
	}
	if a.Runner.Online() {
		return errors.New("the server is running; stop it first (bonghos server stop)")
	}
	inst, err := requireActive(a)
	if err != nil {
		return err
	}
	rec, err := a.Backups.Get(id)
	if err != nil {
		return fmt.Errorf("no such backup: %s", id)
	}
	fmt.Printf("\nRestore %s (%s, created %s) into %s\n", rec.BackupID, rec.BackupType, rec.CreatedAt, inst.Slug)
	fmt.Printf("Scope: %s\n", scope)
	fmt.Println("An emergency pre-restore backup of the current state will be created first.")
	fmt.Print("\nType RESTORE to continue: ")
	if readLine() != "RESTORE" {
		fmt.Println("Cancelled.")
		return nil
	}
	fmt.Println("Creating emergency pre-restore backup …")
	if _, err := a.RunBackup(context.Background(), inst, backup.TypeFull, "offline", "emergency-pre-restore", 0); err != nil {
		return fmt.Errorf("emergency backup failed, refusing to restore: %w", err)
	}
	fmt.Println("Restoring …")
	if err := a.Backups.Restore(rec, inst.AbsoluteDir(a.Home), scope); err != nil {
		return err
	}
	fmt.Println("Restore complete. Start the server with: bonghos server start")
	return nil
}
