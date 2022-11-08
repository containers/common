//go:build linux || solaris || darwin || freebsd
// +build linux solaris darwin freebsd

package lockfile

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/containers/storage/pkg/system"
	"golang.org/x/sys/unix"
)

type lockfile struct {
	// rwMutex serializes concurrent reader-writer acquisitions in the same process space
	rwMutex *sync.RWMutex
	// stateMutex is used to synchronize concurrent accesses to the state below
	stateMutex *sync.Mutex
	counter    int64
	file       string
	fd         uintptr
	lw         []byte // "last writer"-unique value valid as of the last .Touch() or .Modified(), generated by newLastWriterID()
	locktype   int16
	locked     bool
	ro         bool
}

const lastWriterIDSize = 64    // This must be the same as len(stringid.GenerateRandomID)
var lastWriterIDCounter uint64 // Private state for newLastWriterID

// newLastWriterID returns a new "last writer" ID.
// The value must be different on every call, and also differ from values
// generated by other processes.
func newLastWriterID() []byte {
	// The ID is (PID, time, per-process counter, random)
	// PID + time represents both a unique process across reboots,
	// and a specific time within the process; the per-process counter
	// is an extra safeguard for in-process concurrency.
	// The random part disambiguates across process namespaces
	// (where PID values might collide), serves as a general-purpose
	// extra safety, _and_ is used to pad the output to lastWriterIDSize,
	// because other versions of this code exist and they don't work
	// efficiently if the size of the value changes.
	pid := os.Getpid()
	tm := time.Now().UnixNano()
	counter := atomic.AddUint64(&lastWriterIDCounter, 1)

	res := make([]byte, lastWriterIDSize)
	binary.LittleEndian.PutUint64(res[0:8], uint64(tm))
	binary.LittleEndian.PutUint64(res[8:16], counter)
	binary.LittleEndian.PutUint32(res[16:20], uint32(pid))
	if n, err := cryptorand.Read(res[20:lastWriterIDSize]); err != nil || n != lastWriterIDSize-20 {
		panic(err) // This shouldn't happen
	}

	return res
}

// openLock opens the file at path and returns the corresponding file
// descriptor. The path is opened either read-only or read-write,
// depending on the value of ro argument.
//
// openLock will create the file and its parent directories,
// if necessary.
func openLock(path string, ro bool) (fd int, err error) {
	flags := unix.O_CLOEXEC | os.O_CREATE
	if ro {
		flags |= os.O_RDONLY
	} else {
		flags |= os.O_RDWR
	}
	fd, err = unix.Open(path, flags, 0o644)
	if err == nil {
		return fd, nil
	}

	// the directory of the lockfile seems to be removed, try to create it
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return fd, fmt.Errorf("creating locker directory: %w", err)
		}

		return openLock(path, ro)
	}

	return fd, &os.PathError{Op: "open", Path: path, Err: err}
}

// createLockerForPath returns a Locker object, possibly (depending on the platform)
// working inter-process and associated with the specified path.
//
// This function will be called at most once for each path value within a single process.
//
// If ro, the lock is a read-write lock and the returned Locker should correspond to the
// “lock for reading” (shared) operation; otherwise, the lock is either an exclusive lock,
// or a read-write lock and Locker should correspond to the “lock for writing” (exclusive) operation.
//
// WARNING:
// - The lock may or MAY NOT be inter-process.
// - There may or MAY NOT be an actual object on the filesystem created for the specified path.
// - Even if ro, the lock MAY be exclusive.
func createLockerForPath(path string, ro bool) (Locker, error) {
	// Check if we can open the lock.
	fd, err := openLock(path, ro)
	if err != nil {
		return nil, err
	}
	unix.Close(fd)

	locktype := unix.F_WRLCK
	if ro {
		locktype = unix.F_RDLCK
	}
	return &lockfile{
		stateMutex: &sync.Mutex{},
		rwMutex:    &sync.RWMutex{},
		file:       path,
		lw:         newLastWriterID(),
		locktype:   int16(locktype),
		locked:     false,
		ro:         ro}, nil
}

// lock locks the lockfile via FCTNL(2) based on the specified type and
// command.
func (l *lockfile) lock(lType int16) {
	lk := unix.Flock_t{
		Type:   lType,
		Whence: int16(unix.SEEK_SET),
		Start:  0,
		Len:    0,
	}
	switch lType {
	case unix.F_RDLCK:
		l.rwMutex.RLock()
	case unix.F_WRLCK:
		l.rwMutex.Lock()
	default:
		panic(fmt.Sprintf("attempted to acquire a file lock of unrecognized type %d", lType))
	}
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()
	if l.counter == 0 {
		// If we're the first reference on the lock, we need to open the file again.
		fd, err := openLock(l.file, l.ro)
		if err != nil {
			panic(err)
		}
		l.fd = uintptr(fd)

		// Optimization: only use the (expensive) fcntl syscall when
		// the counter is 0.  In this case, we're either the first
		// reader lock or a writer lock.
		for unix.FcntlFlock(l.fd, unix.F_SETLKW, &lk) != nil {
			time.Sleep(10 * time.Millisecond)
		}
	}
	l.locktype = lType
	l.locked = true
	l.counter++
}

// Lock locks the lockfile as a writer.  Panic if the lock is a read-only one.
func (l *lockfile) Lock() {
	if l.ro {
		panic("can't take write lock on read-only lock file")
	} else {
		l.lock(unix.F_WRLCK)
	}
}

// LockRead locks the lockfile as a reader.
func (l *lockfile) RLock() {
	l.lock(unix.F_RDLCK)
}

// Unlock unlocks the lockfile.
func (l *lockfile) Unlock() {
	l.stateMutex.Lock()
	if !l.locked {
		// Panic when unlocking an unlocked lock.  That's a violation
		// of the lock semantics and will reveal such.
		panic("calling Unlock on unlocked lock")
	}
	l.counter--
	if l.counter < 0 {
		// Panic when the counter is negative.  There is no way we can
		// recover from a corrupted lock and we need to protect the
		// storage from corruption.
		panic(fmt.Sprintf("lock %q has been unlocked too often", l.file))
	}
	if l.counter == 0 {
		// We should only release the lock when the counter is 0 to
		// avoid releasing read-locks too early; a given process may
		// acquire a read lock multiple times.
		l.locked = false
		// Close the file descriptor on the last unlock, releasing the
		// file lock.
		unix.Close(int(l.fd))
	}
	if l.locktype == unix.F_RDLCK {
		l.rwMutex.RUnlock()
	} else {
		l.rwMutex.Unlock()
	}
	l.stateMutex.Unlock()
}

func (l *lockfile) AssertLocked() {
	// DO NOT provide a variant that returns the value of l.locked.
	//
	// If the caller does not hold the lock, l.locked might nevertheless be true because another goroutine does hold it, and
	// we can’t tell the difference.
	//
	// Hence, this “AssertLocked” method, which exists only for sanity checks.

	// Don’t even bother with l.stateMutex: The caller is expected to hold the lock, and in that case l.locked is constant true
	// with no possible writers.
	// If the caller does not hold the lock, we are violating the locking/memory model anyway, and accessing the data
	// without the lock is more efficient for callers, and potentially more visible to lock analysers for incorrect callers.
	if !l.locked {
		panic("internal error: lock is not held by the expected owner")
	}
}

func (l *lockfile) AssertLockedForWriting() {
	// DO NOT provide a variant that returns the current lock state.
	//
	// The same caveats as for AssertLocked apply equally.

	l.AssertLocked()
	// Like AssertLocked, don’t even bother with l.stateMutex.
	if l.locktype != unix.F_WRLCK {
		panic("internal error: lock is not held for writing")
	}
}

// Touch updates the lock file with the UID of the user.
func (l *lockfile) Touch() error {
	l.stateMutex.Lock()
	if !l.locked || (l.locktype != unix.F_WRLCK) {
		panic("attempted to update last-writer in lockfile without the write lock")
	}
	defer l.stateMutex.Unlock()
	l.lw = newLastWriterID()
	n, err := unix.Pwrite(int(l.fd), l.lw, 0)
	if err != nil {
		return err
	}
	if n != len(l.lw) {
		return unix.ENOSPC
	}
	return nil
}

// Modified indicates if the lockfile has been updated since the last time it
// was loaded.
func (l *lockfile) Modified() (bool, error) {
	l.stateMutex.Lock()
	if !l.locked {
		panic("attempted to check last-writer in lockfile without locking it first")
	}
	defer l.stateMutex.Unlock()
	currentLW := make([]byte, lastWriterIDSize)
	n, err := unix.Pread(int(l.fd), currentLW, 0)
	if err != nil {
		return true, err
	}
	// It is important to handle the partial read case, because
	// the initial size of the lock file is zero, which is a valid
	// state (no writes yet)
	currentLW = currentLW[:n]
	oldLW := l.lw
	l.lw = currentLW
	return !bytes.Equal(currentLW, oldLW), nil
}

// IsReadWriteLock indicates if the lock file is a read-write lock.
func (l *lockfile) IsReadWrite() bool {
	return !l.ro
}

// TouchedSince indicates if the lock file has been touched since the specified time
func (l *lockfile) TouchedSince(when time.Time) bool {
	st, err := system.Fstat(int(l.fd))
	if err != nil {
		return true
	}
	mtim := st.Mtim()
	touched := time.Unix(mtim.Unix())
	return when.Before(touched)
}