//go:build linux

package backup

import "syscall"

func storageFreeSpace(path string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0
	}
	return int64(st.Bavail) * st.Bsize
}
