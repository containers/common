package ssh

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	// Test adding ssh port
	dst, uri, err := Validate(nil, "ssh://testhost", 0, "")
	require.Nil(t, err)
	require.Equal(t, dst.URI, "ssh://testhost:22")
	require.Equal(t, dst.URI, uri.String())
	dst, _, err = Validate(nil, "ssh://testhost", 22022, "")
	require.Nil(t, err)
	require.Equal(t, dst.URI, "ssh://testhost:22022")

	// Test adding user
	dst, _, err = Validate(url.User("root"), "ssh://testhost", 0, "")
	require.Nil(t, err)
	require.Equal(t, dst.URI, "ssh://root@testhost:22")

	// Test adding identity
	dst, _, err = Validate(nil, "ssh://testhost", 0, "/path/to/sshkey")
	require.Nil(t, err)
	require.Equal(t, dst.Identity, "/path/to/sshkey")

	// Test that the URI path is preserved (#1551)
	dst, _, err = Validate(nil, "ssh://testhost/run/podman/podman.sock", 0, "")
	require.Nil(t, err)
	require.Equal(t, dst.URI, "ssh://testhost:22/run/podman/podman.sock")
	dst, _, err = Validate(nil, "ssh://testhost/var/run/podman/podman.sock", 0, "")
	require.Nil(t, err)
	require.Equal(t, dst.URI, "ssh://testhost:22/var/run/podman/podman.sock")
}
