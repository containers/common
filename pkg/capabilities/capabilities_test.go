package capabilities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllCapabilities(t *testing.T) {
	caps := AllCapabilities()
	assert.True(t, len(caps) > 0)
	err := ValidateCapabilities(caps)
	require.Nil(t, err)
}

func TestBoundingCapabilities(t *testing.T) {
	caps, err := BoundingSet()
	require.Nil(t, err)
	assert.True(t, len(caps) > 0)
}

func TestMergeCapabilitiesDropVerify(t *testing.T) {
	adds := []string{"CAP_SYS_ADMIN", "CAP_SETUID"}
	drops := []string{"CAP_NET_ADMIN", "cap_chown"}
	base := []string{"CHOWN"}
	caps, err := MergeCapabilities(base, adds, drops)
	require.Nil(t, err)
	assert.Equal(t, []string{"CAP_SYS_ADMIN", "CAP_SETUID"}, caps)
}

func TestMergeCapabilitiesDropAddConflict(t *testing.T) {
	adds := []string{"CAP_SYS_ADMIN", "NET_ADMIN"}
	drops := []string{"CAP_NET_ADMIN", "cap_chown"}
	base := []string{"CHOWN"}
	_, err := MergeCapabilities(base, adds, drops)
	assert.Error(t, err)
}

func TestMergeCapabilitiesDrop(t *testing.T) {
	adds := []string{"CAP_SYS_ADMIN"}
	drops := []string{"CAP_NET_ADMIN", "cap_chown"}
	base := []string{"CHOWN"}
	caps, err := MergeCapabilities(base, adds, drops)
	require.Nil(t, err)
	assert.Equal(t, []string{"CAP_SYS_ADMIN"}, caps)
}

func TestMergeCapabilitiesDropAll(t *testing.T) {
	adds := []string{"CAP_SYS_ADMIN", "CAP_NET_ADMIN", "CAP_CHOWN"}
	drops := []string{"all"}
	base := []string{"CAP_SETUID"}
	caps, err := MergeCapabilities(base, adds, drops)
	require.Nil(t, err)
	assert.Equal(t, caps, adds)
}

func TestMergeCapabilitiesAddAll(t *testing.T) {
	base := []string{"CAP_SYS_ADMIN", "CAP_NET_ADMIN", "CAP_CHOWN"}
	adds := []string{"all"}
	drops := []string{}
	caps, err := MergeCapabilities(base, adds, drops)
	require.Nil(t, err)
	allCaps, err := BoundingSet()
	require.Nil(t, err)
	assert.Equal(t, caps, allCaps)
}

func TestNormalizeCapabilities(t *testing.T) {
	strSlice := []string{"SYS_ADMIN", "net_admin", "CAP_CHOWN"}
	caps, err := NormalizeCapabilities(strSlice)
	require.Nil(t, err)
	err = ValidateCapabilities(caps)
	require.Nil(t, err)
	strSlice = []string{"no_ADMIN", "net_admin", "CAP_CHMOD"}
	_, err = NormalizeCapabilities(strSlice)
	assert.Error(t, err)
}

func TestValidateCapabilities(t *testing.T) {
	strSlice := []string{"CAP_SYS_ADMIN", "CAP_NET_ADMIN"}
	err := ValidateCapabilities(strSlice)
	require.Nil(t, err)
}

func TestValidateCapabilitieBadCapabilities(t *testing.T) {
	strSlice := []string{"CAP_SYS_ADMIN", "NO_ADMIN"}
	err := ValidateCapabilities(strSlice)
	assert.Error(t, err)
}
