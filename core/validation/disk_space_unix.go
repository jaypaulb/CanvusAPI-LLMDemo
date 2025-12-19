//go:build !windows

package validation

import (
	"syscall"
)

// getDiskSpace returns total and free bytes for the filesystem containing path.
// Unix/Linux implementation using syscall.Statfs.
func getDiskSpace(path string) (total int64, free int64, err error) {
	var stat syscall.Statfs_t
	err = syscall.Statfs(path, &stat)
	if err != nil {
		return 0, 0, err
	}

	// Total space = block size * total blocks
	total = int64(stat.Blocks) * int64(stat.Bsize)

	// Free space = block size * available blocks (for non-root users)
	// Use Bavail instead of Bfree to get space available to unprivileged users
	free = int64(stat.Bavail) * int64(stat.Bsize)

	return total, free, nil
}
