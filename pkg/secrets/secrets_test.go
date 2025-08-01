package secrets

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

var drivertype = "file"

func setup(t *testing.T) (manager *SecretsManager, opts map[string]string) {
	testpath := t.TempDir()
	manager, err := NewManager(testpath)
	require.NoError(t, err)
	return manager, map[string]string{"path": testpath}
}

func TestAddSecretAndLookupData(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
		Metadata:   map[string]string{"immutable": "true"},
		Labels: map[string]string{
			"foo":     "bar",
			"another": "label",
		},
	}

	id1, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	_, err = manager.lookupSecret("mysecret")
	require.NoError(t, err)

	s, data, err := manager.LookupSecretData("mysecret")
	require.NoError(t, err)
	if !bytes.Equal(data, []byte("mydata")) {
		t.Errorf("error: secret data not equal")
	}
	if val, ok := s.Metadata["immutable"]; !ok || val != "true" {
		t.Errorf("error: no metadata")
	}
	if val, ok := s.Labels["foo"]; !ok || val != "bar" {
		t.Errorf("error: label incorrect")
	}
	if len(s.Labels) != 2 {
		t.Errorf("error: incorrect number of labels")
	}
	if s.CreatedAt != s.UpdatedAt {
		t.Errorf("error: secret CreatedAt should equal UpdatedAt when first created")
	}

	_, err = manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.Error(t, err)

	storeOpts.Replace = true
	id2, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)
	if id1 == id2 {
		t.Errorf("error: secret id after Replace should be different")
	}

	s, _, err = manager.LookupSecretData("mysecret")
	require.NoError(t, err)
	if s.CreatedAt.Equal(s.UpdatedAt) {
		t.Errorf("error: secret CreatedAt should not equal UpdatedAt after a Replace")
	}

	_, _, err = manager.LookupSecretData(id2)
	require.NoError(t, err)

	_, _, err = manager.LookupSecretData(id1)
	require.Error(t, err)
}

func TestAddSecretName(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
	}

	longstring := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for _, value := range []string{"a", "user@mail.com", longstring[:253]} {
		// test one char secret name
		_, err := manager.Store(value, []byte("mydata"), drivertype, storeOpts)
		require.NoError(t, err)

		_, err = manager.lookupSecret(value)
		require.NoError(t, err)
	}

	for _, value := range []string{"", "chocolate,vanilla", "file/path", "foo=bar", "bad\000Null", longstring[:254]} {
		_, err := manager.Store(value, []byte("mydata"), drivertype, storeOpts)
		require.Error(t, err)
	}
}

func TestAddMultipleSecrets(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
	}

	id, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	id2, err := manager.Store("mysecret2", []byte("mydata2"), drivertype, storeOpts)
	require.NoError(t, err)

	secrets, err := manager.List()
	require.NoError(t, err)
	require.Len(t, secrets, 2)

	_, err = manager.lookupSecret("mysecret")
	require.NoError(t, err)

	_, err = manager.lookupSecret("mysecret2")
	require.NoError(t, err)

	_, data, err := manager.LookupSecretData(id)
	require.NoError(t, err)
	if !bytes.Equal(data, []byte("mydata")) {
		t.Errorf("error: secret data not equal")
	}

	_, data2, err := manager.LookupSecretData(id2)
	require.NoError(t, err)
	if !bytes.Equal(data2, []byte("mydata2")) {
		t.Errorf("error: secret data not equal")
	}
}

func TestAddSecretDupName(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
	}

	_, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	_, err = manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.Error(t, err)

	storeOpts.Replace = true
	_, err = manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	storeOpts.IgnoreIfExists = true
	_, err = manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.Error(t, err)

	storeOpts.Replace = false
	_, err = manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)
}

func TestAddReplaceSecretName(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
		Replace:    true,
	}

	_, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	_, err = manager.Store("mysecret", []byte("mydata.diff"), drivertype, storeOpts)
	require.NoError(t, err)

	_, data, err := manager.LookupSecretData("mysecret")
	require.NoError(t, err)
	require.Equal(t, string(data), "mydata.diff")

	_, err = manager.Store("nonexistingsecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	storeOpts.Replace = false
	_, err = manager.Store("nonexistingsecret", []byte("newdata"), drivertype, storeOpts)
	require.Error(t, err)

	_, err = manager.Delete("nonexistingsecret")
	require.NoError(t, err)
}

func TestAddSecretPrefix(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
	}

	// If the randomly generated secret id is something like "abcdeiuoergnadufigh"
	// we should still allow someone to store a secret with the name "abcd" or "a"
	secretID, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	_, err = manager.Store(secretID[0:5], []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)
}

func TestRemoveSecret(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
	}

	_, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	_, err = manager.lookupSecret("mysecret")
	require.NoError(t, err)

	_, err = manager.Delete("mysecret")
	require.NoError(t, err)

	_, err = manager.lookupSecret("mysecret")
	require.Error(t, err)

	_, _, err = manager.LookupSecretData("mysecret")
	require.Error(t, err)
}

func TestRemoveSecretNoExist(t *testing.T) {
	manager, _ := setup(t)

	_, err := manager.Delete("mysecret")
	require.Error(t, err)
}

func TestLookupAllSecrets(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
	}

	id, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	// inspect using secret name
	lookup, err := manager.Lookup("mysecret")
	require.NoError(t, err)
	require.Equal(t, lookup.ID, id)
}

func TestInspectSecretId(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
	}

	id, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)

	_, err = manager.lookupSecret("mysecret")
	require.NoError(t, err)

	// inspect using secret id
	lookup, err := manager.Lookup(id)
	require.NoError(t, err)
	require.Equal(t, lookup.ID, id)

	// inspect using id prefix
	short := id[0:5]
	lookupshort, err := manager.Lookup(short)
	require.NoError(t, err)
	require.Equal(t, lookupshort.ID, id)
}

func TestInspectSecretBogus(t *testing.T) {
	manager, _ := setup(t)

	_, err := manager.Lookup("bogus")
	require.Error(t, err)
}

func TestSecretList(t *testing.T) {
	manager, opts := setup(t)

	storeOpts := StoreOptions{
		DriverOpts: opts,
	}

	_, err := manager.Store("mysecret", []byte("mydata"), drivertype, storeOpts)
	require.NoError(t, err)
	_, err = manager.Store("mysecret2", []byte("mydata2"), drivertype, storeOpts)
	require.NoError(t, err)

	allSecrets, err := manager.List()
	require.NoError(t, err)
	require.Len(t, allSecrets, 2)
}
