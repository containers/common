package parse

import (
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDeviceParser verifies the given device strings is parsed correctly
func TestDeviceParser(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Devices is only supported on Linux")
	}

	// Test defaults
	src, dest, permissions, err := Device("/dev/foo")
	assert.NoError(t, err)
	assert.Equal(t, src, "/dev/foo")
	assert.Equal(t, dest, "/dev/foo")
	assert.Equal(t, permissions, "rwm")

	// Test defaults, different dest
	src, dest, permissions, err = Device("/dev/foo:/dev/bar")
	assert.NoError(t, err)
	assert.Equal(t, src, "/dev/foo")
	assert.Equal(t, dest, "/dev/bar")
	assert.Equal(t, permissions, "rwm")

	// Test fully specified
	src, dest, permissions, err = Device("/dev/foo:/dev/bar:rm")
	assert.NoError(t, err)
	assert.Equal(t, src, "/dev/foo")
	assert.Equal(t, dest, "/dev/bar")
	assert.Equal(t, permissions, "rm")

	// Test device, permissions
	src, dest, permissions, err = Device("/dev/foo:rm")
	assert.NoError(t, err)
	assert.Equal(t, src, "/dev/foo")
	assert.Equal(t, dest, "/dev/foo")
	assert.Equal(t, permissions, "rm")

	// test bogus permissions
	_, _, _, err = Device("/dev/fuse1:BOGUS") //nolint
	assert.Error(t, err)

	_, _, _, err = Device("") //nolint
	assert.Error(t, err)

	_, _, _, err = Device("/dev/foo:/dev/bar:rm:") //nolint
	assert.Error(t, err)

	_, _, _, err = Device("/dev/foo::rm") //nolint
	assert.Error(t, err)
}

func TestIsValidDeviceMode(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Devices is only supported on Linux")
	}
	assert.False(t, isValidDeviceMode("BOGUS"))
	assert.False(t, isValidDeviceMode("rwx"))
	assert.True(t, isValidDeviceMode("r"))
	assert.True(t, isValidDeviceMode("rw"))
	assert.True(t, isValidDeviceMode("rm"))
	assert.True(t, isValidDeviceMode("rwm"))
}

func TestDeviceFromPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Devices is only supported on Linux")
	}
	// Path is valid
	info, err := os.Stat("/dev/null")
	assert.NoError(t, err)

	var UID uint32
	var GID uint32
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		UID = stat.Uid
		GID = stat.Gid
	}
	dev, err := DeviceFromPath("/dev/null")
	assert.NoError(t, err)
	assert.Equal(t, len(dev), 1)
	assert.Equal(t, dev[0].Major, int64(1))
	assert.Equal(t, dev[0].Minor, int64(3))
	assert.Equal(t, string(dev[0].Permissions), "rwm")
	assert.Equal(t, dev[0].Uid, UID)
	assert.Equal(t, dev[0].Gid, GID)

	// Path does not exists
	_, err = DeviceFromPath("/dev/BOGUS")
	assert.Error(t, err)

	// Path is a directory of devices
	_, err = DeviceFromPath("/dev/pts")
	assert.NoError(t, err)

	// path of directory has no device
	_, err = DeviceFromPath("/etc/passwd")
	assert.Error(t, err)
}
