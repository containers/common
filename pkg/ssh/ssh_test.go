package ssh

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// these tests cannot check for "true" functionality
// in order to do that, you need two machines and a place to connect to/from
// these will error but we can check the error message to make sure it is an ssh error message
// not one for a segfault or parsing error

func TestCreate(t *testing.T) {
	options := ConnectionCreateOptions{
		Port:    22,
		Path:    "localhost",
		Name:    "testing",
		Socket:  "/run/user/foo/podman/podman.sock",
		Default: false,
	}
	err := Create(&options, NativeMode)
	// exit status 255 is what you get when ssh is not enabled or the connection failed
	// this means up to that point, everything worked
	require.Error(t, err, "exit status 255")

	err = Create(&options, GolangMode)
	// the error with golang should be nil, we want this to work if we are given a socket path
	// that is the current podman behavior
	require.Nil(t, err)
}

func TestExec(t *testing.T) {
	options := ConnectionExecOptions{
		Port: 22,
		Host: "localhost",
		Args: []string{"ls", "/"},
	}

	_, err := Exec(&options, NativeMode)
	// exit status 255 is what you get when ssh is not enabled or the connection failed
	// this means up to that point, everything worked
	require.Error(t, err, "exit status 255")

	_, err = Exec(&options, GolangMode)
	require.Error(t, err, "failed to connect: ssh: handshake failed: ssh: disconnect, reason 2: Too many authentication failures")
}

func TestDial(t *testing.T) {
	options := ConnectionDialOptions{
		Port: 22,
		Host: "localhost",
	}

	_, err := Dial(&options, NativeMode)
	// exit status 255 is what you get when ssh is not enabled or the connection failed
	// this means up to that point, everything worked
	require.Error(t, err, "exit status 255")

	_, err = Dial(&options, GolangMode)
	require.Error(t, err, "failed to connect: ssh: handshake failed: ssh: disconnect, reason 2: Too many authentication failures")

	// Test again without specifying sshd port, and code should default to port 22
	options = ConnectionDialOptions{
		Host: "localhost",
	}

	_, err = Dial(&options, NativeMode)
	// exit status 255 is what you get when ssh is not enabled or the connection failed
	// this means up to that point, everything worked
	require.Error(t, err, "exit status 255")

	_, err = Dial(&options, GolangMode)
	require.Error(t, err, "failed to connect: ssh: handshake failed: ssh: disconnect, reason 2: Too many authentication failures")
}

func TestScp(t *testing.T) {
	f, err := os.CreateTemp("", "")
	require.Nil(t, err)

	defer os.Remove(f.Name())

	options := ConnectionScpOptions{
		User:        &url.Userinfo{},
		Source:      f.Name(),
		Destination: "localhost:/does/not/exist",
		Port:        22,
	}

	_, err = Scp(&options, NativeMode)
	// exit status 255 is what you get when ssh is not enabled or the connection failed
	// this means up to that point, everything worked
	require.Error(t, err, "exit status 255")

	_, err = Scp(&options, GolangMode)
	require.Error(t, err, "failed to connect: ssh: handshake failed: ssh: disconnect, reason 2: Too many authentication failures")
}
