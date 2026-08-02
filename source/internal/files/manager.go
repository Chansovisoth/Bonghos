package files

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Chansovisoth/Bonghos/internal/security"
)

// Manager exposes constrained file operations inside one server root.
// Every path is canonically validated; the manager can never reach outside.
type Manager struct {
	Root          string
	MaxEditBytes  int64 // refuse text-editing larger files
	MaxUploadFile int64
}

type Entry struct {
	Name     string `json:"name"`
	Path     string `json:"path"` // relative to root
	IsDir    bool   `json:"is_dir"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

func (m *Manager) resolve(rel string) (string, error) {
	rel = strings.TrimPrefix(strings.TrimSpace(rel), "/")
	abs, err := security.SafeJoin(m.Root, rel)
	if err != nil {
		return "", err
	}
	return security.EvalWithinRoot(m.Root, abs)
}

func (m *Manager) List(rel string) ([]Entry, error) {
	dir, err := m.resolve(rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Entry{
			Name:     e.Name(),
			Path:     filepath.ToSlash(filepath.Join(rel, e.Name())),
			IsDir:    e.IsDir(),
			Size:     info.Size(),
			Modified: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (m *Manager) ReadText(rel string) (string, error) {
	p, err := m.resolve(rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("is a directory")
	}
	if m.MaxEditBytes > 0 && info.Size() > m.MaxEditBytes {
		return "", fmt.Errorf("file too large to edit (%d bytes)", info.Size())
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", errors.New("binary files cannot be edited as text")
	}
	return string(data), nil
}

// WriteText saves a text file atomically after creating a timestamped backup
// for important files.
func (m *Manager) WriteText(rel, content string) error {
	p, err := m.resolve(rel)
	if err != nil {
		return err
	}
	if m.MaxEditBytes > 0 && int64(len(content)) > m.MaxEditBytes {
		return errors.New("content too large")
	}
	if info, err := os.Stat(p); err == nil && !info.IsDir() {
		backupImportant(p)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return atomicWrite(p, []byte(content), 0o644)
}

var importantNames = map[string]bool{
	"server.properties": true, "eula.txt": true, "user_jvm_args.txt": true,
	"whitelist.json": true, "ops.json": true, "banned-players.json": true,
	"banned-ips.json": true,
}

func backupImportant(p string) {
	if !importantNames[filepath.Base(p)] {
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}
	stamp := nowStamp()
	os.WriteFile(p+".bonghos-backup-"+stamp, data, 0o644)
}

func (m *Manager) Mkdir(rel string) error {
	p, err := m.resolve(rel)
	if err != nil {
		return err
	}
	return os.MkdirAll(p, 0o755)
}

func (m *Manager) Rename(fromRel, toRel string) error {
	from, err := m.resolve(fromRel)
	if err != nil {
		return err
	}
	to, err := m.resolve(toRel)
	if err != nil {
		return err
	}
	if _, err := os.Stat(to); err == nil {
		return errors.New("destination already exists")
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.Rename(from, to)
}

func (m *Manager) Delete(rel string) error {
	p, err := m.resolve(rel)
	if err != nil {
		return err
	}
	if filepath.Clean(p) == filepath.Clean(m.Root) {
		return errors.New("refusing to delete the server root")
	}
	return os.RemoveAll(p)
}

func (m *Manager) Open(rel string) (*os.File, os.FileInfo, error) {
	p, err := m.resolve(rel)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	if info.IsDir() {
		f.Close()
		return nil, nil, errors.New("is a directory")
	}
	return f, info, nil
}

func (m *Manager) SaveUpload(rel string, r io.Reader) (int64, error) {
	p, err := m.resolve(rel)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return 0, err
	}
	limit := m.MaxUploadFile
	if limit <= 0 {
		limit = 4 << 30
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, io.LimitReader(r, limit+1))
	cerr := f.Close()
	if n > limit {
		os.Remove(p)
		return n, errors.New("upload exceeds size limit")
	}
	if err != nil {
		return n, err
	}
	return n, cerr
}

// Search finds filenames containing q (case-insensitive), capped at 500 hits.
func (m *Manager) Search(q string) ([]Entry, error) {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil, nil
	}
	var out []Entry
	err := filepath.Walk(m.Root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if len(out) >= 500 {
			return errors.New("done")
		}
		if strings.Contains(strings.ToLower(info.Name()), q) {
			rel, _ := filepath.Rel(m.Root, p)
			out = append(out, Entry{
				Name: info.Name(), Path: filepath.ToSlash(rel),
				IsDir: info.IsDir(), Size: info.Size(),
				Modified: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
			})
		}
		return nil
	})
	if err != nil && err.Error() != "done" {
		return out, err
	}
	return out, nil
}

// DirSize computes the total regular-file size of a tree.
func DirSize(root string) int64 {
	var total int64
	filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// MoveTree moves src to dst atomically when possible (same filesystem),
// otherwise copies then removes.
func MoveTree(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := CopyTree(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

// CopyTree copies a directory tree preserving modes; symlinks are copied as
// links (already validated during extraction).
func CopyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		switch {
		case info.IsDir():
			return os.MkdirAll(out, info.Mode().Perm())
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			os.Remove(out)
			return os.Symlink(target, out)
		case info.Mode().IsRegular():
			return copyFile(p, out, info.Mode().Perm())
		}
		return nil
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".bonghos-tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	tmp.Chmod(mode)
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func nowStamp() string {
	return time.Now().UTC().Format("2006-01-02_15-04-05")
}
