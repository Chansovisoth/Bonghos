// Package files implements safe archive extraction and constrained file
// management. Every entry path is validated against traversal, absolute
// paths, unsafe links and decompression bombs before touching disk.
package files

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/klauspost/compress/zstd"
)

// Limits bound extraction to protect against decompression bombs.
type Limits struct {
	MaxBytes int64 // total decompressed bytes
	MaxFiles int64
	// FreeSpaceReserve is how many bytes must remain free on the destination
	// filesystem. Extraction stops before consuming it, so a large pack cannot
	// fill the disk the running server depends on. Zero disables the check.
	FreeSpaceReserve int64
}

var (
	ErrTraversal   = errors.New("archive entry escapes destination (path traversal)")
	ErrAbsolute    = errors.New("archive entry uses an absolute path")
	ErrUnsafeLink  = errors.New("archive contains an unsafe link")
	ErrTooLarge    = errors.New("archive exceeds the decompressed size limit")
	ErrTooMany     = errors.New("archive exceeds the file count limit")
	ErrUnsupported = errors.New("unsupported archive format")
)

// DetectFormat sniffs the archive type from magic bytes, falling back to the
// filename. The original filename is never trusted as a directory name.
func DetectFormat(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	switch {
	case n >= 4 && head[0] == 'P' && head[1] == 'K':
		return "zip", nil
	case n >= 2 && head[0] == 0x1f && head[1] == 0x8b:
		return "tar.gz", nil
	case n >= 6 && string(head[:6]) == "\xfd7zXZ\x00":
		return "tar.xz", nil
	case n >= 4 && head[0] == 0x28 && head[1] == 0xb5 && head[2] == 0x2f && head[3] == 0xfd:
		return "tar.zst", nil
	case n >= 6 && string(head[:6]) == "7z\xbc\xaf\x27\x1c":
		return "7z", nil
	case n >= 7 && string(head[:7]) == "Rar!\x1a\x07\x00",
		n >= 8 && string(head[:8]) == "Rar!\x1a\x07\x01\x00":
		return "rar", nil
	}
	// tar has magic at offset 257
	if _, err := f.Seek(257, io.SeekStart); err == nil {
		magic := make([]byte, 5)
		if _, err := io.ReadFull(f, magic); err == nil && string(magic) == "ustar" {
			return "tar", nil
		}
	}
	// fall back to extension
	name := strings.ToLower(path)
	for ext, format := range map[string]string{
		".zip": "zip", ".tar": "tar", ".tar.gz": "tar.gz", ".tgz": "tar.gz",
		".tar.xz": "tar.xz", ".tar.zst": "tar.zst", ".7z": "7z", ".rar": "rar",
	} {
		if strings.HasSuffix(name, ext) {
			return format, nil
		}
	}
	return "", ErrUnsupported
}

// safeEntryPath validates an archive entry name and returns its destination.
func safeEntryPath(dest, name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/") // normalize Windows separators
	if name == "" {
		return "", ErrTraversal
	}
	if strings.HasPrefix(name, "/") || filepath.IsAbs(name) ||
		(len(name) >= 2 && name[1] == ':') { // C:\ style
		return "", ErrAbsolute
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrTraversal
	}
	out := filepath.Join(dest, clean)
	rel, err := filepath.Rel(dest, out)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrTraversal
	}
	return out, nil
}

// Extract unpacks archive at src into dest with full safety validation.
// Format is auto-detected. progress (may be nil) receives decompressed bytes.
// ErrDiskSpace reports that extraction would consume the configured reserve.
var ErrDiskSpace = errors.New("not enough free disk space to extract this archive safely")

// FreeSpace returns the bytes available to an unprivileged user on the
// filesystem holding dir. A zero result means the check is unavailable.
func FreeSpace(dir string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0
	}
	return int64(st.Bavail) * int64(st.Bsize)
}

// checkSpace reports whether extracting `incoming` more bytes into dir would
// eat into the reserve. Unavailable statistics never block extraction.
func checkSpace(dir string, incoming int64, lim Limits) error {
	if lim.FreeSpaceReserve <= 0 {
		return nil
	}
	free := FreeSpace(dir)
	if free == 0 {
		return nil
	}
	if free < lim.FreeSpaceReserve+incoming {
		return ErrDiskSpace
	}
	return nil
}

func Extract(src, dest string, lim Limits, progress func(bytes int64)) error {
	format, err := DetectFormat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	// Check before starting: a compressed archive typically expands several
	// times over, so its own size is a conservative lower bound.
	if st, serr := os.Stat(src); serr == nil {
		if err := checkSpace(dest, st.Size(), lim); err != nil {
			return err
		}
	}
	switch format {
	case "zip":
		return extractZip(src, dest, lim, progress)
	case "tar":
		return extractTarStream(src, dest, lim, progress, func(r io.Reader) (io.Reader, error) { return r, nil })
	case "tar.gz":
		return extractTarStream(src, dest, lim, progress, func(r io.Reader) (io.Reader, error) {
			return gzip.NewReader(r)
		})
	case "tar.xz":
		return extractExternal(src, dest, lim, "tar", "--no-same-owner", "-xJf", src, "-C", dest)
	case "tar.zst":
		return extractTarStream(src, dest, lim, progress, func(r io.Reader) (io.Reader, error) {
			zr, err := zstd.NewReader(r)
			if err != nil {
				return nil, err
			}
			return zr.IOReadCloser(), nil
		})
	case "7z":
		if _, err := exec.LookPath("7z"); err != nil {
			return fmt.Errorf("%w: install p7zip-full for .7z support", ErrUnsupported)
		}
		return extractExternal(src, dest, lim, "7z", "x", "-y", "-o"+dest, src)
	case "rar":
		bin := ""
		for _, c := range []string{"unrar", "unar"} {
			if _, err := exec.LookPath(c); err == nil {
				bin = c
				break
			}
		}
		if bin == "" {
			return fmt.Errorf("%w: install unrar or unar for .rar support", ErrUnsupported)
		}
		if bin == "unrar" {
			return extractExternal(src, dest, lim, "unrar", "x", "-y", src, dest+string(filepath.Separator))
		}
		return extractExternal(src, dest, lim, "unar", "-quiet", "-o", dest, src)
	}
	return ErrUnsupported
}

func extractZip(src, dest string, lim Limits, progress func(int64)) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	var total, count int64
	for _, f := range zr.File {
		count++
		if lim.MaxFiles > 0 && count > lim.MaxFiles {
			return ErrTooMany
		}
		out, err := safeEntryPath(dest, f.Name)
		if err != nil {
			return fmt.Errorf("%w: %s", err, f.Name)
		}
		mode := f.Mode()
		if mode&os.ModeSymlink != 0 {
			// resolve link target safely (must stay inside dest)
			rc, err := f.Open()
			if err != nil {
				return err
			}
			target, err := io.ReadAll(io.LimitReader(rc, 4096))
			rc.Close()
			if err != nil {
				return err
			}
			if err := safeSymlink(dest, out, string(target)); err != nil {
				return err
			}
			continue
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		n, err := writeLimited(out, rc, mode.Perm(), lim.MaxBytes-total)
		rc.Close()
		total += n
		if progress != nil {
			progress(total)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarStream(src, dest string, lim Limits, progress func(int64),
	wrap func(io.Reader) (io.Reader, error)) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := wrap(f)
	if err != nil {
		return err
	}
	if c, ok := r.(io.Closer); ok {
		defer c.Close()
	}
	tr := tar.NewReader(r)
	var total, count int64
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		count++
		if lim.MaxFiles > 0 && count > lim.MaxFiles {
			return ErrTooMany
		}
		out, err := safeEntryPath(dest, hdr.Name)
		if err != nil {
			return fmt.Errorf("%w: %s", err, hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := safeSymlink(dest, out, hdr.Linkname); err != nil {
				return err
			}
		case tar.TypeLink:
			// hard link target must live inside the destination
			targetOut, err := safeEntryPath(dest, hdr.Linkname)
			if err != nil {
				return fmt.Errorf("hard-link escape: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			if err := os.Link(targetOut, out); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			n, err := writeLimited(out, tr, os.FileMode(hdr.Mode).Perm(), lim.MaxBytes-total)
			total += n
			if progress != nil {
				progress(total)
			}
			if err != nil {
				return err
			}
		default:
			// char/block devices, fifos etc. are skipped deliberately
		}
	}
}

// extractExternal shells out (argument arrays only, never shell strings) for
// formats without a safe pure-Go reader, then post-validates the result.
func extractExternal(src, dest string, lim Limits, bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s extraction failed: %w", bin, err)
	}
	return postValidateTree(dest, lim)
}

// postValidateTree walks an extracted tree removing links that escape dest
// and enforcing limits (defense for external extractors).
func postValidateTree(dest string, lim Limits) error {
	var total, count int64
	return filepath.Walk(dest, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		count++
		if lim.MaxFiles > 0 && count > lim.MaxFiles {
			return ErrTooMany
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			resolved := target
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(filepath.Dir(p), target)
			}
			rel, err := filepath.Rel(dest, filepath.Clean(resolved))
			if err != nil || rel == ".." || strings.HasPrefix(rel, "..") || filepath.IsAbs(target) {
				os.Remove(p) // remove the unsafe link
				return nil
			}
		}
		if info.Mode().IsRegular() {
			total += info.Size()
			if lim.MaxBytes > 0 && total > lim.MaxBytes {
				return ErrTooLarge
			}
		}
		return nil
	})
}

func safeSymlink(dest, linkPath, target string) error {
	if filepath.IsAbs(target) {
		return fmt.Errorf("%w: absolute symlink %s", ErrUnsafeLink, target)
	}
	resolved := filepath.Join(filepath.Dir(linkPath), target)
	rel, err := filepath.Rel(dest, filepath.Clean(resolved))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: %s -> %s", ErrUnsafeLink, linkPath, target)
	}
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return err
	}
	os.Remove(linkPath)
	return os.Symlink(target, linkPath)
}

func writeLimited(path string, r io.Reader, mode os.FileMode, remaining int64) (int64, error) {
	if remaining <= 0 {
		return 0, ErrTooLarge
	}
	if mode == 0 {
		mode = 0o644
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, io.LimitReader(r, remaining+1))
	cerr := f.Close()
	if n > remaining {
		os.Remove(path)
		return n, ErrTooLarge
	}
	if err != nil {
		return n, err
	}
	return n, cerr
}

// FindServerRoot identifies the actual server root inside an extracted
// staging directory, unwrapping a single unnecessary outer directory when the
// real root can be identified safely by server markers.
func FindServerRoot(staging string) (string, error) {
	markers := []string{
		"server.properties", "eula.txt", "run.sh", "start.sh", "startserver.sh",
		"server-start.sh", "start-server.sh", "mods", "libraries", "user_jvm_args.txt",
	}
	hasMarker := func(dir string) bool {
		for _, m := range markers {
			if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
				return true
			}
		}
		return false
	}
	if hasMarker(staging) {
		return staging, nil
	}
	// unwrap up to 2 levels of single-directory nesting
	dir := staging
	for depth := 0; depth < 2; depth++ {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return "", err
		}
		var subdirs []string
		hasFiles := false
		for _, e := range entries {
			if e.IsDir() {
				subdirs = append(subdirs, e.Name())
			} else if !strings.HasPrefix(e.Name(), ".") {
				hasFiles = true
			}
		}
		if hasFiles || len(subdirs) != 1 {
			break
		}
		dir = filepath.Join(dir, subdirs[0])
		if hasMarker(dir) {
			return dir, nil
		}
	}
	// fall back to staging root itself
	return staging, nil
}
