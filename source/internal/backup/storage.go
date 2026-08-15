package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ValidateStorageRoot rejects filesystem roots and nested moves. A backup
// storage move eventually removes the old tree, so both paths must be exact,
// ordinary directories rather than broad or overlapping targets.
func ValidateStorageRoot(current, target string) error {
	current = filepath.Clean(current)
	target = filepath.Clean(target)
	if !filepath.IsAbs(current) || !filepath.IsAbs(target) {
		return errors.New("backup storage paths must be absolute")
	}
	if current == target {
		return errors.New("new backup storage is already the current location")
	}
	if filepath.Dir(current) == current || filepath.Dir(target) == target {
		return errors.New("a filesystem root cannot be used as backup storage")
	}
	if withinPath(current, target) || withinPath(target, current) {
		return errors.New("backup storage locations cannot contain one another")
	}
	return nil
}

func withinPath(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// DirectoryEmpty reports whether a directory contains no entries. A missing
// directory is considered empty so first-time storage configuration remains
// straightforward.
func DirectoryEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// StorageEmpty reports whether a storage tree has no files. Empty layout
// directories created during setup do not make an otherwise fresh backup
// location "in use".
func StorageEmpty(path string) (bool, error) {
	st, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("backup storage cannot be a symbolic link")
	}
	if !st.IsDir() {
		return false, errors.New("backup storage is not a directory")
	}
	empty := true
	err = filepath.WalkDir(path, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != path && !entry.IsDir() {
			empty = false
			return fs.SkipAll
		}
		return nil
	})
	return empty, err
}

// CopyStorageVerified copies an entire backup tree into a new, nonexistent
// directory and hashes every regular file after copying. Symlinks and special
// files are rejected so an external storage path cannot escape containment.
// The caller changes configuration only after this function succeeds.
func CopyStorageVerified(current, target string) error {
	if err := ValidateStorageRoot(current, target); err != nil {
		return err
	}
	st, err := os.Lstat(current)
	if err != nil {
		return fmt.Errorf("reading current backup storage: %w", err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return errors.New("current backup storage cannot be a symbolic link")
	}
	if !st.IsDir() {
		return errors.New("current backup storage is not a directory")
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return errors.New("destination already exists; choose a new path")
		}
		return err
	}
	if err := os.MkdirAll(target, st.Mode().Perm()); err != nil {
		return err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			_ = os.RemoveAll(target)
		}
	}()
	err = filepath.WalkDir(current, func(src string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(current, src)
		if err != nil || rel == "." {
			return err
		}
		dst := filepath.Join(target, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("backup storage contains unsupported symlink: %s", rel)
		}
		if entry.IsDir() {
			return os.Mkdir(dst, info.Mode().Perm())
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("backup storage contains unsupported special file: %s", rel)
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
		sourceHash, sourceSize, err := fileChecksum(src)
		if err != nil {
			return err
		}
		targetHash, targetSize, err := fileChecksum(dst)
		if err != nil {
			return err
		}
		if sourceHash != targetHash || sourceSize != targetSize {
			return fmt.Errorf("verification failed after copying %s", rel)
		}
		return os.Chmod(dst, info.Mode().Perm())
	})
	if err != nil {
		return err
	}
	succeeded = true
	return nil
}

// RemoveStorage removes the exact old storage directory after a verified copy
// and configuration switch. ValidateStorageRoot must be called first by the
// move workflow; this final guard still refuses broad filesystem roots.
func RemoveStorage(path string) error {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) || filepath.Dir(path) == path {
		return errors.New("refusing to remove an unsafe backup storage path")
	}
	return os.RemoveAll(path)
}
