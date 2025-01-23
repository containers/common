package detach

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	smallBytes = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	bigBytes   = []byte(strings.Repeat("0F", 32*1024+30))
)

func newCustomReader(buf *bytes.Buffer, readsize uint) *customReader {
	return &customReader{
		inner:    buf,
		readsize: readsize,
	}
}

type customReader struct {
	inner    *bytes.Buffer
	readsize uint
}

func (c *customReader) Read(p []byte) (n int, err error) {
	return c.inner.Read(p[:min(int(c.readsize), len(p))])
}

func FuzzCopy(f *testing.F) {
	for _, i := range []uint{1, 2, 3, 5, 10, 100, 200, 1000, 1024, 32 * 1024} {
		f.Add(i)
	}

	f.Fuzz(func(t *testing.T, readSize uint) {
		// 0 is not a valid read size
		if readSize == 0 {
			t.Skip()
		}

		tests := []struct {
			name         string
			from         []byte
			expected     []byte
			expectDetach bool
			keys         []byte
		}{
			{
				name:     "small copy",
				from:     smallBytes,
				expected: smallBytes,
				keys:     nil,
			},
			{
				name:     "small copy with detach keys",
				from:     smallBytes,
				expected: smallBytes,
				keys:     []byte{'A', 'B'},
			},
			{
				name:     "big copy",
				from:     bigBytes,
				expected: bigBytes,
				keys:     nil,
			},
			{
				name:     "big copy with detach keys",
				from:     bigBytes,
				expected: bigBytes,
				keys:     []byte{'A', 'B'},
			},
			{
				name:         "simple detach 1 key",
				from:         append(smallBytes, 'A'),
				expected:     smallBytes,
				expectDetach: true,
				keys:         []byte{'A'},
			},
			{
				name:         "simple detach 2 keys",
				from:         append(smallBytes, 'A', 'B'),
				expected:     smallBytes,
				expectDetach: true,
				keys:         []byte{'A', 'B'},
			},
			{
				name:         "simple detach 3 keys",
				from:         append(smallBytes, 'A', 'B', 'C'),
				expected:     smallBytes,
				expectDetach: true,
				keys:         []byte{'A', 'B', 'C'},
			},
			{
				name:         "detach early",
				from:         append(smallBytes, 'A', 'B', 'B', 'A'),
				expected:     smallBytes,
				expectDetach: true,
				keys:         []byte{'A', 'B'},
			},
			{
				name:         "detach with partial match",
				from:         append(smallBytes, 'A', 'A', 'A', 'B'),
				expected:     append(smallBytes, 'A', 'A'),
				expectDetach: true,
				keys:         []byte{'A', 'B'},
			},
			{
				name:         "big buffer detach with partial match",
				from:         append(bigBytes, 'A', 'A', 'A', 'B'),
				expected:     append(bigBytes, 'A', 'A'),
				expectDetach: true,
				keys:         []byte{'A', 'B'},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dst := &bytes.Buffer{}
				src := newCustomReader(bytes.NewBuffer(tt.from), readSize)
				written, err := Copy(dst, src, tt.keys)
				if tt.expectDetach {
					assert.ErrorIs(t, err, ErrDetach)
				} else {
					assert.NoError(t, err)
				}
				assert.Equal(t, dst.Len(), int(written), "bytes written matches buffer")
				assert.Equal(t, tt.expected, dst.Bytes(), "buffer matches")
			})
		}
	})
}
