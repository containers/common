package rootlessnetns

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_refCount(t *testing.T) {
	tests := []struct {
		name    string
		content string
		inc     int
		want    int
	}{
		{
			name: "init counter",
			inc:  1,
			want: 1,
		},
		{
			name: "simple add",
			inc:  5,
			want: 5,
		},
		{
			name:    "add multiple with content",
			content: "0",
			inc:     5,
			want:    5,
		},
		{
			name:    "add multiple with high number content",
			content: "5500",
			inc:     2,
			want:    5502,
		},
		{
			name:    "simple dec",
			content: "5",
			inc:     -5,
			want:    0,
		},
		{
			name:    "dec negative should not go below 0",
			content: "0",
			inc:     -5,
			want:    0,
		},
		{
			name:    "dec multiple with high number content",
			content: "9800",
			inc:     -100,
			want:    9700,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, refCountFile)
			if tt.content != "" {
				err := os.WriteFile(file, []byte(tt.content), 0o700)
				assert.NoError(t, err, "write file error")
			}

			got, err := refCount(dir, tt.inc)
			assert.NoError(t, err, "refCount() error")
			assert.Equal(t, tt.want, got, "counter is equal")
			content, err := os.ReadFile(file)
			assert.NoError(t, err, "read file error")
			assert.Equal(t, strconv.Itoa(tt.want), string(content), "file content after refCount()")
		})
	}
}
