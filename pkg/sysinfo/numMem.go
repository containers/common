// +build amd64, arm, freebsd

package sysinfo

import (
	"golang.org/x/sys/unix"
	"os"
	"unsafe"
)

// NUMANodeCount queries the system for the count of Memory Nodes available
// for use to this process.
func NUMANodeCount() int {
	// Gets the affinity mask for a process: The very one invoking this function.
	pid := os.Getpid()
	var mask [1024 / 64]uintptr
	_, _, err := unix.RawSyscall(unix.SYS_NUMA_GETAFFINITY, pid, uintptr(len(mask)*8), uintptr(unsafe.Pointer(&mask[0])))
	if err != 0 {
		return 0
	}

	// For every available thread a bit is set in the mask.
	nmem := 0
	for _, e := range mask {
		if e == 0 {
			continue
		}
		nmem += int(popcnt(uint64(e)))
	}
	return nmem
}
