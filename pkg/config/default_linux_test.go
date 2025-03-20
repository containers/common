package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultMaskedPaths(t *testing.T) {
	maskedPaths, err := getMaskedPaths()
	assert.NoError(t, err)

	root := "/sys/devices/system/cpu"

	entries, err := os.ReadDir(root)
	assert.NoError(t, err)

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "cpu") {
			continue
		}
		path := filepath.Join(root, entry.Name(), "thermal_throttle")
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			assert.NoError(t, err)
		}
		assert.Contains(t, maskedPaths, path)
	}
}
