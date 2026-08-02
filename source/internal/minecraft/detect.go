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
	reXms      = regexp.MustCompile(`-Xms(\S+)`)
	reXmx      = regexp.MustCompile(`-Xmx(\S+)`)
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
}

// DetectJVMConfig inspects the selected startup script, sourced files and
// known argument files, returning the best editable source found.
func DetectJVMConfig(root, startupRel string) (*JVMConfig, error) {
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
		if reXms.MatchString(line) && xms != "" {
			lines[i] = reXms.ReplaceAllString(line, "-Xms"+xms)
			replacedXms = true
		}
		if reXmx.MatchString(line) && xmx != "" {
			lines[i] = reXmx.ReplaceAllString(line, "-Xmx"+xmx)
			replacedXmx = true
		}
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
