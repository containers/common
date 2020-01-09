package rootless

import (
	"fmt"
	"os"
	"os/user"

	"github.com/containers/storage"
	"github.com/pkg/errors"
)

func TryJoinPauseProcess(pausePidPath string) (bool, int, error) {
	if _, err := os.Stat(pausePidPath); err != nil {
		return false, -1, nil
	}

	became, ret, err := TryJoinFromFilePaths("", false, []string{pausePidPath})
	if err == nil {
		return became, ret, err
	}

	// It could not join the pause process, let's lock the file before trying to delete it.
	pidFileLock, err := storage.GetLockfile(pausePidPath)
	if err != nil {
		// The file was deleted by another process.
		if os.IsNotExist(err) {
			return false, -1, nil
		}
		return false, -1, errors.Wrapf(err, "error acquiring lock on %s", pausePidPath)
	}

	pidFileLock.Lock()
	defer func() {
		if pidFileLock.Locked() {
			pidFileLock.Unlock()
		}
	}()

	// Now the pause PID file is locked.  Try to join once again in case it changed while it was not locked.
	became, ret, err = TryJoinFromFilePaths("", false, []string{pausePidPath})
	if err != nil {
		// It is still failing.  We can safely remove it.
		os.Remove(pausePidPath)
		return false, -1, nil
	}
	return became, ret, err
}

// HomeDir returns the home directory for the current user.
func HomeDir() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		usr, err := user.LookupId(fmt.Sprintf("%d", GetRootlessUID()))
		if err != nil {
			return "", errors.Wrapf(err, "unable to resolve HOME directory")
		}
		home = usr.HomeDir
	}
	return home, nil
}
