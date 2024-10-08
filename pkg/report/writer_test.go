package report

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWriter(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello\tWorld", "Hello:World"},
		{"Be\tGood", "Be::Good"},
	}

	var buf bytes.Buffer
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			w, err := NewWriter(&buf, 4, 8, 1, ':', 0)
			assert.NoError(t, err)

			_, err = w.Write([]byte(tc.input))
			assert.NoError(t, err)
			w.Flush()
			assert.Equal(t, tc.expected, buf.String())
		})
		buf.Reset()
	}
}

func TestNewWriterDefault(t *testing.T) {
	var buf bytes.Buffer
	w, err := NewWriterDefault(&buf)
	assert.NoError(t, err)
	_, err = w.Write([]byte("a46001bfea3a4172b46c173101208244\tRandom\tuuid"))
	assert.NoError(t, err)
	w.Flush()
	assert.Equal(t, "a46001bfea3a4172b46c173101208244  Random      uuid", buf.String())
}
