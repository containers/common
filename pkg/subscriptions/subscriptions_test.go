package subscriptions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadAllAndSaveTo(t *testing.T) {
	const testMode = os.FileMode(0o700)

	rootDir := t.TempDir()
	childDir := filepath.Join(rootDir, "child")
	err := os.Mkdir(childDir, testMode)
	assert.NoError(t, err, "mkdir child")

	filePath := "child/file"
	err = os.WriteFile(filepath.Join(rootDir, filePath), []byte("test"), testMode)
	assert.NoError(t, err, "write file")

	data, err := readAll(rootDir, "", testMode)
	assert.NoError(t, err, "readAll")
	assert.Len(t, data, 1, "readAll should return one result")

	tmpDir := t.TempDir()
	err = data[0].saveTo(tmpDir)
	assert.NoError(t, err, "saveTo()")

	assert.FileExists(t, filepath.Join(tmpDir, filePath), "file exists at correct location")
}
