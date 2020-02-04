package config

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// isCgroup2UnifiedMode returns whether we are running in cgroup2 mode.
func isCgroup2UnifiedMode() (isUnified bool, isUnifiedErr error) {
	cgroupRoot := "/sys/fs/cgroup"

	var st syscall.Statfs_t
	if err := syscall.Statfs(cgroupRoot, &st); err != nil {
		isUnified, isUnifiedErr = false, err
	} else {
		isUnified, isUnifiedErr = st.Type == unix.CGROUP2_SUPER_MAGIC, nil
	}
	return
}
