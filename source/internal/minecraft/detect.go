// Package minecraft implements modpack-aware detection: startup scripts,
// JVM configuration sources, log parsing and player administration files.
package minecraft

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ----- startup script detection ---------------------------------------------

var knownStartupNames = map[string]int{
	"run.sh": 90, "start.sh": 85, "startserver.sh": 85,
	"server-start.sh": 80, "start-server.sh": 80,
}

var (
	reJavaCmd     = regexp.MustCompile(`(?m)\bjava\b|\$JAVA|"?\$\{?JAVA`)
	reUnixArgs    = regexp.MustCompile(`@(user_jvm_args\.txt|libraries/[^\s"']*unix_args\.txt)`)
	reForge       = regexp.MustCompile(`(?i)forge|neoforge`)
	reFabric      = regexp.MustCompile(`(?i)fabric-server|fabric_installer|fabric\.jar`)
	reQuilt       = regexp.MustCompile(`(?i)quilt`)
	reSourced     = regexp.MustCompile(`(?m)^\s*(?:\.|source)\s+([^\s;]+)`)
	reInteractive = regexp.MustCompile(`(?m)^\s*(read\s+-[nprs]|read\s+-r\s+-p|pause\b)|Press any key`)
)

// StartupCandidate is a ranked launch-script candidate.
type StartupCandidate struct {
	Path        string   `json:"path"` // relative to server root
	Score       int      `json:"score"`
	HasJava     bool     `json:"has_java"`
	Modloader   string   `json:"modloader"`
	UsesArgFile string   `json:"uses_arg_file,omitempty"`
	Interactive []string `json:"interactive_lines,omitempty"` // "file:line: text"
}

// DetectStartupScripts searches the server root (bounded depth) for launch
// scripts, inspects their content, and returns ranked candidates.
func DetectStartupScripts(root string, maxDepth int) ([]StartupCandidate, error) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	rootAbs, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	var candidates []StartupCandidate
	err = filepath.Walk(rootAbs, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(rootAbs, p)
		if rerr != nil {
			return nil
		}
		depth := strings.Count(rel, string(filepath.Separator))
		if info.IsDir() {
			if depth >= maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, lerr := filepath.EvalSymlinks(p)
			if lerr != nil || !strings.HasPrefix(resolved, rootAbs+string(filepath.Separator)) {
				return nil // reject links escaping the project
			}
		}
		if !strings.HasSuffix(info.Name(), ".sh") {
			return nil
		}
		c := inspectScript(rootAbs, rel)
		if c != nil {
			candidates = append(candidates, *c)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	return candidates, nil
}

func inspectScript(root, rel string) *StartupCandidate {
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil || len(data) > 1<<20 {
		return nil
	}
	body := string(data)
	c := StartupCandidate{Path: filepath.ToSlash(rel)}
	if base := filepath.Base(rel); knownStartupNames[base] > 0 {
		c.Score += knownStartupNames[base]
	}
	if reJavaCmd.MatchString(body) {
		c.HasJava = true
		c.Score += 40
	}
	if m := reUnixArgs.FindStringSubmatch(body); m != nil {
		c.UsesArgFile = m[1]
		c.Score += 20
	}
	switch {
	case reForge.MatchString(body):
		c.Modloader = "forge"
		if strings.Contains(strings.ToLower(body), "neoforge") {
			c.Modloader = "neoforge"
		}
		c.Score += 10
	case reFabric.MatchString(body):
		c.Modloader = "fabric"
		c.Score += 10
	case reQuilt.MatchString(body):
		c.Modloader = "quilt"
		c.Score += 10
	}
	// interactive prompt detection with file:line reporting
	for i, line := range strings.Split(body, "\n") {
		if reInteractive.MatchString(line) {
			c.Interactive = append(c.Interactive,
				fmt.Sprintf("%s:%d: %s", c.Path, i+1, strings.TrimSpace(line)))
		}
	}
	// follow sourced scripts one level for java hints
	for _, m := range reSourced.FindAllStringSubmatch(body, 5) {
		sub := filepath.Join(root, filepath.Dir(rel), strings.Trim(m[1], `"'`))
		if resolved, err := filepath.EvalSymlinks(sub); err == nil &&
			strings.HasPrefix(resolved, root+string(filepath.Separator)) {
			if sd, err := os.ReadFile(resolved); err == nil && reJavaCmd.Match(sd) {
				c.HasJava = true
				c.Score += 10
			}
		}
	}
	if c.Score == 0 {
		return nil
	}
	return &c
}

// PatchInteractive rewrites known interactive patterns to non-blocking forms.
// It returns the proposed new content and a unified-style diff preview; the
// caller must show the diff, back up the original and get confirmation before
// writing anything.
func PatchInteractive(content string) (patched string, diff []string) {
	lines := strings.Split(content, "\n")
	out := make([]string, len(lines))
	copy(out, lines)
	for i, line := range lines {
		if reInteractive.MatchString(line) {
			replacement := "# [bonghos] interactive prompt disabled for supervised operation: " + strings.TrimSpace(line)
			out[i] = replacement
			diff = append(diff, fmt.Sprintf("-%s", line), fmt.Sprintf("+%s", replacement))
		}
	}
	return strings.Join(out, "\n"), diff
}

// ----- JVM configuration detection -------------------------------------------

var jvmVarNames = []string{
	"JAVA_ARGS", "JVM_ARGS", "JAVA_OPTS", "JVM_OPTS",
	"XMS", "XMX", "MIN_RAM", "MAX_RAM", "MIN_MEMORY", "MAX_MEMORY", "MEMORY", "RAM",
}

var (
	// Match only the memory value itself. A greedy \S+ swallows a trailing
	// quote in JAVA_ARGS="-Xmx4G -Xms4G", so rewriting the value corrupted
	// the shell assignment and the new setting never took effect.
	reXms      = regexp.MustCompile(`-Xms(\d+[kKmMgGtT]?)`)
	reXmx      = regexp.MustCompile(`-Xmx(\d+[kKmMgGtT]?)`)
	reAssign   = regexp.MustCompile(`(?m)^\s*(?:export\s+)?([A-Z_]+)=["']?([^"'\n#]*)`)
	memValueRE = regexp.MustCompile(`^\d+[kKmMgG]?$`)
)

// JVMConfig describes detected JVM memory settings and their source.
type JVMConfig struct {
	SourceFile string `json:"source_file"` // relative path controlling the values
	SourceKind string `json:"source_kind"` // arg_file | variable | script
	Variable   string `json:"variable,omitempty"`
	Xms        string `json:"xms"`
	Xmx        string `json:"xmx"`
	ExtraArgs  string `json:"extra_args"`
	Editable   bool   `json:"editable"`
	// Note explains why a source cannot be edited, or what owns it.
	Note string `json:"note,omitempty"`
}

// detectOwningVariable looks for a JVM variable assignment in the startup
// script family that takes precedence over a generated argument file.
func detectOwningVariable(root, startupRel string) *JVMConfig {
	if startupRel == "" {
		return nil
	}
	for _, f := range scriptFamily(root, startupRel) {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		for _, m := range reAssign.FindAllStringSubmatch(string(data), -1) {
			name, value := m[1], strings.TrimSpace(m[2])
			if !contains(jvmVarNames, name) {
				continue
			}
			if !strings.Contains(value, "-Xm") {
				continue // not the memory-bearing assignment
			}
			// Only take over when an argument file would otherwise win and
			// that file is regenerated; otherwise the normal order applies.
			generated := false
			for _, af := range []string{"user_jvm_args.txt", "jvm_args.txt", "java_args.txt"} {
				if _, err := os.Stat(filepath.Join(root, af)); err != nil {
					continue
				}
				if argFileIsGenerated(root, startupRel, af) {
					generated = true
					break
				}
			}
			if !generated {
				return nil
			}
			cfg := &JVMConfig{
				SourceFile: filepath.ToSlash(f), SourceKind: "variable",
				Variable: name, Editable: true,
				Note: "This pack regenerates its argument file at launch, so " +
					name + " here is the setting that actually takes effect.",
			}
			parseJVMArgs(value, cfg)
			return cfg
		}
	}
	return nil
}

// generatedArgFile matches a startup script writing an argument file itself,
// which means anything the panel edits there is discarded on the next start.
// ServerPackCreator packs do exactly this: variables.txt holds JAVA_ARGS and
// start.sh regenerates user_jvm_args.txt from it on every run.
var generatedArgFile = regexp.MustCompile(
	`(?m)(?:>>?|\btee\b)\s*"?[^"\n]*?(user_jvm_args\.txt|jvm_args\.txt|java_args\.txt)`)

// argFileIsGenerated reports whether the startup script (or a script it
// sources) rewrites the given argument file at launch.
func argFileIsGenerated(root, startupRel, argFile string) bool {
	if startupRel == "" {
		return false
	}
	base := filepath.Base(argFile)
	for _, f := range scriptFamily(root, startupRel) {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		body := string(data)
		for _, m := range generatedArgFile.FindAllStringSubmatch(body, -1) {
			if m[1] == base {
				return true
			}
		}
		// Some packs say so in a comment rather than in obvious redirection.
		lower := strings.ToLower(body)
		if strings.Contains(lower, strings.ToLower(base)) &&
			(strings.Contains(lower, "regenerat") || strings.Contains(lower, "overwrit") ||
				strings.Contains(lower, "do not edit") || strings.Contains(lower, "don't edit") ||
				strings.Contains(lower, "will be replaced")) {
			return true
		}
	}
	return false
}

// scriptFamily returns the startup script plus any scripts it sources.
func scriptFamily(root, startupRel string) []string {
	files := []string{startupRel}
	data, err := os.ReadFile(filepath.Join(root, startupRel))
	if err != nil {
		return files
	}
	for _, m := range reSourced.FindAllStringSubmatch(string(data), 8) {
		files = append(files, filepath.ToSlash(
			filepath.Join(filepath.Dir(startupRel), strings.Trim(m[1], `"'`))))
	}
	// ServerPackCreator keeps its settings beside the script even when the
	// script reads it without a `source` line.
	for _, known := range []string{"variables.txt", "settings.cfg"} {
		cand := filepath.ToSlash(filepath.Join(filepath.Dir(startupRel), known))
		if _, err := os.Stat(filepath.Join(root, cand)); err == nil {
			files = append(files, cand)
		}
	}
	return files
}

// DetectJVMConfig inspects the selected startup script, sourced files and
// known argument files, returning the best editable source found.
func DetectJVMConfig(root, startupRel string) (*JVMConfig, error) {
	// Priority 0: a variable that owns the launch settings. When the startup
	// script regenerates the argument file, the argument file is an output,
	// not a source: editing it looks like it worked and is silently reverted
	// on the next restart.
	if cfg := detectOwningVariable(root, startupRel); cfg != nil {
		return cfg, nil
	}

	// Priority 1: dedicated JVM argument files.
	argFiles := []string{"user_jvm_args.txt", "jvm_args.txt", "java_args.txt"}
	// include any @file references from the startup script
	if startupRel != "" {
		if data, err := os.ReadFile(filepath.Join(root, startupRel)); err == nil {
			for _, m := range reUnixArgs.FindAllStringSubmatch(string(data), 3) {
				argFiles = append([]string{m[1]}, argFiles...)
			}
		}
	}
	for _, af := range argFiles {
		p := filepath.Join(root, af)
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			rootAbs, _ := filepath.EvalSymlinks(root)
			if !strings.HasPrefix(resolved, rootAbs+string(filepath.Separator)) {
				continue
			}
			data, err := os.ReadFile(resolved)
			if err != nil {
				continue
			}
			cfg := &JVMConfig{
				SourceFile: filepath.ToSlash(af), SourceKind: "arg_file", Editable: true,
			}
			parseJVMArgs(string(data), cfg)
			if argFileIsGenerated(root, startupRel, af) {
				// Nothing better was found, so report it, but do not let the
				// panel write to a file the pack rewrites at launch.
				cfg.Editable = false
				cfg.Note = "This file is regenerated by the startup script on every run, " +
					"so changes made here are lost. Edit the launch settings the script reads instead."
			}
			// unix_args.txt from Forge installers is pack-owned; editing xms/xmx
			// there is still safe line-wise, but user_jvm_args.txt is preferred.
			return cfg, nil
		}
	}
	// Priority 2: variable assignment inside the startup script or sourced files.
	if startupRel != "" {
		files := []string{startupRel}
		if data, err := os.ReadFile(filepath.Join(root, startupRel)); err == nil {
			for _, m := range reSourced.FindAllStringSubmatch(string(data), 5) {
				files = append(files, filepath.ToSlash(filepath.Join(filepath.Dir(startupRel), strings.Trim(m[1], `"'`))))
			}
		}
		for _, f := range files {
			data, err := os.ReadFile(filepath.Join(root, f))
			if err != nil {
				continue
			}
			body := string(data)
			for _, m := range reAssign.FindAllStringSubmatch(body, -1) {
				name, value := m[1], strings.TrimSpace(m[2])
				if !contains(jvmVarNames, name) {
					continue
				}
				cfg := &JVMConfig{
					SourceFile: f, SourceKind: "variable", Variable: name, Editable: true,
				}
				parseJVMArgs(value, cfg)
				if cfg.Xms == "" && memValueRE.MatchString(value) &&
					(name == "XMS" || name == "MIN_RAM" || name == "MIN_MEMORY") {
					cfg.Xms = value
				}
				if cfg.Xmx == "" && memValueRE.MatchString(value) &&
					(name == "XMX" || name == "MAX_RAM" || name == "MAX_MEMORY" || name == "MEMORY" || name == "RAM") {
					cfg.Xmx = value
				}
				if cfg.Xms != "" || cfg.Xmx != "" || cfg.ExtraArgs != "" {
					return cfg, nil
				}
			}
			// raw -Xms/-Xmx inside the script
			if reXms.MatchString(body) || reXmx.MatchString(body) {
				cfg := &JVMConfig{SourceFile: f, SourceKind: "script", Editable: false}
				parseJVMArgs(body, cfg)
				return cfg, nil
			}
		}
	}
	// Priority 3: scan shallow files containing -Xms/-Xmx.
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if e.IsDir() || e.Name() == filepath.Base(startupRel) {
			continue
		}
		if info, err := e.Info(); err != nil || info.Size() > 1<<20 {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, e.Name()))
		if err != nil {
			continue
		}
		if reXms.Match(data) || reXmx.Match(data) {
			cfg := &JVMConfig{SourceFile: e.Name(), SourceKind: "script", Editable: false}
			parseJVMArgs(string(data), cfg)
			return cfg, nil
		}
	}
	return &JVMConfig{SourceKind: "none"}, nil
}

func parseJVMArgs(s string, cfg *JVMConfig) {
	if m := reXms.FindStringSubmatch(s); m != nil {
		cfg.Xms = m[1]
	}
	if m := reXmx.FindStringSubmatch(s); m != nil {
		cfg.Xmx = m[1]
	}
	var extra []string
	for _, tok := range strings.Fields(s) {
		if strings.HasPrefix(tok, "-X") || strings.HasPrefix(tok, "-XX") {
			if !strings.HasPrefix(tok, "-Xms") && !strings.HasPrefix(tok, "-Xmx") {
				extra = append(extra, tok)
			}
		}
	}
	cfg.ExtraArgs = strings.Join(extra, " ")
}

// UpdateJVMArgFile rewrites -Xms/-Xmx lines inside a dedicated argument file
// while preserving comments and unrelated content. Returns new content.
func UpdateJVMArgFile(content, xms, xmx string) string {
	lines := strings.Split(content, "\n")
	replacedXms, replacedXmx := false, false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Both substitutions must build on each other. Applying the second to
		// the original line discarded the first, which silently lost the Xms
		// change whenever both appeared together, as in
		// JAVA_ARGS="-Xmx4G -Xms4G".
		updated := line
		if reXms.MatchString(updated) && xms != "" {
			updated = reXms.ReplaceAllString(updated, "-Xms"+xms)
			replacedXms = true
		}
		if reXmx.MatchString(updated) && xmx != "" {
			updated = reXmx.ReplaceAllString(updated, "-Xmx"+xmx)
			replacedXmx = true
		}
		lines[i] = updated
	}
	if !replacedXms && xms != "" {
		lines = append(lines, "-Xms"+xms)
	}
	if !replacedXmx && xmx != "" {
		lines = append(lines, "-Xmx"+xmx)
	}
	return strings.Join(lines, "\n")
}

// ValidateMemory checks Xms <= Xmx and that Xmx leaves headroom for the host.
func ValidateMemory(xms, xmx string, hostTotalBytes int64) error {
	msB, err := parseMem(xms)
	if err != nil {
		return fmt.Errorf("invalid Xms: %w", err)
	}
	mxB, err := parseMem(xmx)
	if err != nil {
		return fmt.Errorf("invalid Xmx: %w", err)
	}
	if msB > mxB {
		return fmt.Errorf("Xms (%s) must not exceed Xmx (%s)", xms, xmx)
	}
	if hostTotalBytes > 0 {
		// Leave at least 1.5 GiB or 10% for Linux and Bonghos.
		reserve := int64(3 << 29)
		if p := hostTotalBytes / 10; p > reserve {
			reserve = p
		}
		if mxB > hostTotalBytes-reserve {
			return fmt.Errorf("Xmx %s leaves too little RAM for Linux and Bonghos (host has %d MiB)",
				xmx, hostTotalBytes>>20)
		}
	}
	return nil
}

func parseMem(v string) (int64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, fmt.Errorf("empty value")
	}
	mult := int64(1)
	switch v[len(v)-1] {
	case 'k', 'K':
		mult = 1 << 10
		v = v[:len(v)-1]
	case 'm', 'M':
		mult = 1 << 20
		v = v[:len(v)-1]
	case 'g', 'G':
		mult = 1 << 30
		v = v[:len(v)-1]
	}
	var n int64
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil || n <= 0 {
		return 0, fmt.Errorf("cannot parse %q", v)
	}
	return n * mult, nil
}

// ----- Java discovery --------------------------------------------------------

type JavaInstall struct {
	Path    string `json:"path"`
	Version string `json:"version"` // e.g. "21"
}

// DiscoverJava finds installed Java executables from PATH and common
// locations, deduplicated by resolved path.
func DiscoverJava() []JavaInstall {
	seen := map[string]bool{}
	var out []JavaInstall
	add := func(p string) {
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil || seen[resolved] {
			return
		}
		seen[resolved] = true
		v := javaVersionOf(resolved)
		if v == "" {
			return
		}
		out = append(out, JavaInstall{Path: resolved, Version: v})
	}
	for _, dir := range strings.Split(os.Getenv("PATH"), ":") {
		p := filepath.Join(dir, "java")
		if st, err := os.Stat(p); err == nil && st.Mode()&0o111 != 0 {
			add(p)
		}
	}
	for _, glob := range []string{
		"/usr/lib/jvm/*/bin/java", "/opt/java/*/bin/java", "/opt/jdk*/bin/java",
	} {
		matches, _ := filepath.Glob(glob)
		for _, m := range matches {
			add(m)
		}
	}
	return out
}

func javaVersionOf(bin string) string {
	// Read version from the release file next to bin/java (fast, no exec).
	release := filepath.Join(filepath.Dir(bin), "..", "release")
	if data, err := os.ReadFile(release); err == nil {
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "JAVA_VERSION=") {
				v := strings.Trim(strings.TrimPrefix(line, "JAVA_VERSION="), `"`)
				if major, _, ok := strings.Cut(v, "."); ok {
					if major == "1" { // 1.8 style
						parts := strings.Split(v, ".")
						if len(parts) > 1 {
							return parts[1]
						}
					}
					return major
				}
				return v
			}
		}
	}
	return "unknown"
}

// ResolveJava maps a java_selection (auto | java | java-17 | /path) to an
// executable path, or an error listing the discovered options.
func ResolveJava(selection string) (string, error) {
	installs := DiscoverJava()
	switch {
	case selection == "" || selection == "auto":
		if len(installs) == 1 {
			return installs[0].Path, nil
		}
		if len(installs) == 0 {
			return "", fmt.Errorf("no Java installation found; install a JDK (e.g. openjdk-21-jre-headless)")
		}
		// prefer the newest version when unambiguous ordering exists
		best := installs[0]
		for _, j := range installs[1:] {
			if j.Version > best.Version {
				best = j
			}
		}
		return best.Path, nil
	case selection == "java":
		for _, j := range installs {
			return j.Path, nil
		}
		return "", fmt.Errorf("java not found on PATH")
	case strings.HasPrefix(selection, "java-"):
		want := strings.TrimPrefix(selection, "java-")
		for _, j := range installs {
			if j.Version == want {
				return j.Path, nil
			}
		}
		return "", fmt.Errorf("Java %s not found; installed: %v", want, installs)
	default:
		if st, err := os.Stat(selection); err == nil && st.Mode()&0o111 != 0 {
			return selection, nil
		}
		return "", fmt.Errorf("java executable %q not found or not executable", selection)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
