package passdriver

import (
	"context"
	"fmt"
	"testing"

	"github.com/containers/common/pkg/secrets/define"
	"github.com/stretchr/testify/require"
)

const gpgTestID = "testing@passdriver"

func setupDriver(t *testing.T) *Driver {
	base := t.TempDir()
	gpghomedir := t.TempDir()

	driver, err := NewDriver(map[string]string{
		"root":       base,
		"key":        gpgTestID,
		"gpghomedir": gpghomedir,
	})
	require.NoError(t, err)

	err = driver.gpg(context.TODO(), nil, nil, "--batch", "--passphrase", "--quick-generate-key", "testing@passdriver")
	require.NoError(t, err)

	return driver
}

func TestStoreAndLookup(t *testing.T) {
	cases := []struct {
		name         string
		key          string
		value        []byte
		expStoreErr  error
		expLookupErr error
	}{
		{
			name:  "store and lookup work for a simple key",
			key:   "simple",
			value: []byte("abc"),
		},
		{
			name:  "store and lookup work for a multiline string",
			key:   "long",
			value: []byte("abc\n123\ndef\n"),
		},
		{
			name:  "store and lookup work for non-utf8 data",
			key:   "long",
			value: []byte{0, 1, 2, 3, 0, 1, 2, 3},
		},
		{
			name:        "storing into a sneaky key fails",
			key:         "../../../sneaky",
			value:       []byte("abc"),
			expStoreErr: define.ErrInvalidKey,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			driver := setupDriver(t)
			err := driver.Store(tc.key, tc.value)
			if tc.expStoreErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expStoreErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				val, err := driver.Lookup(tc.key)
				if tc.expLookupErr != nil {
					require.Error(t, err)
					require.Equal(t, tc.expLookupErr.Error(), err.Error())
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.value, val)
				}
			}
		})
	}
}

func TestLookup(t *testing.T) {
	driver := setupDriver(t)

	// prepare a valid lookup target
	err := driver.Store("valid", []byte("abc"))
	require.NoError(t, err)

	cases := []struct {
		name     string
		key      string
		expValue []byte
		expErr   error
	}{
		{
			name:     "lookup of an existing key works",
			key:      "valid",
			expValue: []byte("abc"),
		},
		{
			name:   "lookup of a non-existing key fails",
			key:    "invalid",
			expErr: fmt.Errorf("invalid: %w", define.ErrNoSuchSecret),
		},
		{
			name:   "lookup of a sneaky key fails",
			key:    "../../../etc/shadow",
			expErr: define.ErrInvalidKey,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			val, err := driver.Lookup(tc.key)
			if tc.expErr == nil {
				require.Equal(t, tc.expValue, val)
			} else {
				require.EqualError(t, err, tc.expErr.Error())
			}
		})
	}
}

func TestList(t *testing.T) {
	driver := setupDriver(t)
	require.NoError(t, driver.Store("a", []byte("abc")))
	require.NoError(t, driver.Store("b", []byte("abc")))
	require.NoError(t, driver.Store("c", []byte("abc")))

	list, err := driver.List()
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c"}, list)
}

func TestDelete(t *testing.T) {
	driver := setupDriver(t)
	require.NoError(t, driver.Store("a", []byte("abc")))

	cases := []struct {
		name   string
		key    string
		expErr error
	}{
		{
			name: "deleting an existing item works",
			key:  "a",
		},
		{
			name:   "deleting an non-existing item fails",
			key:    "wrong",
			expErr: fmt.Errorf("wrong: %w", define.ErrNoSuchSecret),
		},
		{
			name:   "using a sneaky path fails",
			key:    "../../../etc/shadow",
			expErr: define.ErrInvalidKey,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := driver.Delete(tc.key)
			if tc.expErr != nil {
				require.EqualError(t, err, tc.expErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
