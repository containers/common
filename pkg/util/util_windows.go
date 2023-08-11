//go:build windows
// +build windows

package util

import (
	"os"
)

// getRuntimeDir returns the runtime directory
func GetRuntimeDir() (string, error) {
	tmpDir, ok := os.LookupEnv("TEMP")
	if ok {
		return tmpDir, nil
	}

	tmpDir, ok = os.LookupEnv("LOCALAPPDATA")
	if ok {
		tmpDir += `\Temp`
	}

	if !ok {
		tmpDir, ok = os.LookupEnv("USERPROFILE")
		if !ok {
			tmpDir, ok = os.LookupEnv("HOME")
		}
		// Append to either match
		if ok {
			tmpDir += `\AppData\Local\Temp`
		}
	}

	if !ok {
		// Hope for the best
		return `C:\Temp`, nil
	}
	_ = os.MkdirAll(tmpDir, 0o700)

	return tmpDir, nil
}
