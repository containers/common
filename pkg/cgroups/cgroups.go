package cgroups

import (
	"sync"
	"syscall"
)

const (
	cgroupRoot         = "/sys/fs/cgroup"
	_cgroup2SuperMagic = 0x63677270
)

var (
	isUnifiedOnce sync.Once
	isUnified     bool
	isUnifiedErr  error
)

// IsCgroup2UnifiedMode returns whether we are running in cgroup 2 cgroup2 mode.
func IsCgroup2UnifiedMode() (bool, error) {
	isUnifiedOnce.Do(func() {
		var st syscall.Statfs_t
		if err := syscall.Statfs(cgroupRoot, &st); err != nil {
			isUnified, isUnifiedErr = false, err
		} else {
			isUnified, isUnifiedErr = st.Type == _cgroup2SuperMagic, nil
		}
	})
	return isUnified, isUnifiedErr
}
