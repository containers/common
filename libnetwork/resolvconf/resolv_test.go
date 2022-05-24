package resolvconf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

const resolv1 = `nameserver 1.1.1.1
`

const resolv2 = `search example.com
nameserver 1.1.1.1
options edns0
`

func TestNew(t *testing.T) {
	tests := []struct {
		name            string
		baseContent     string
		nameservers     []string
		options         []string
		searches        []string
		ipv6            bool
		hostns          bool
		keepHostServers bool
		want            string
	}{
		{
			name:        "simple resolv.conf",
			baseContent: resolv1,
			want:        resolv1,
		},
		{
			name:        "simple resolv.conf with search and options",
			baseContent: resolv2,
			want:        resolv2,
		},
		{
			name:        "simple resolv.conf with comments",
			baseContent: "#some comment\n" + resolv2,
			want:        resolv2,
		},
		{
			name:        "overwrite default nameservers",
			baseContent: resolv2,
			nameservers: []string{"1.2.3.4", "5.6.7.8"},
			want:        "search example.com\nnameserver 1.2.3.4\nnameserver 5.6.7.8\noptions edns0\n",
		},
		{
			name:        "overwrite default options",
			baseContent: resolv2,
			options:     []string{"ndots:2"},
			want:        "search example.com\nnameserver 1.1.1.1\noptions ndots:2\n",
		},
		{
			name:        "overwrite default searches",
			baseContent: resolv2,
			searches:    []string{"test.com"},
			want:        "search test.com\nnameserver 1.1.1.1\noptions edns0\n",
		},
		{
			name:        "dot in searches should unset all search domains",
			baseContent: resolv2,
			searches:    []string{"."},
			want:        "nameserver 1.1.1.1\noptions edns0\n",
		},
		{
			name:            "dot in searches should unset all search domains even with keepHostServers",
			baseContent:     resolv2,
			searches:        []string{"."},
			keepHostServers: true,
			want:            "nameserver 1.1.1.1\noptions edns0\n",
		},
		{
			name:        "overwrite all",
			baseContent: resolv2,
			nameservers: []string{"1.2.3.4", "5.6.7.8"},
			options:     []string{"ndots:2"},
			searches:    []string{"test.com"},
			want:        "search test.com\nnameserver 1.2.3.4\nnameserver 5.6.7.8\noptions ndots:2\n",
		},
		{
			name:        "overwrite all and unset search",
			baseContent: resolv2,
			nameservers: []string{"1.2.3.4", "5.6.7.8"},
			options:     []string{"ndots:2"},
			searches:    []string{"."},
			want:        "nameserver 1.2.3.4\nnameserver 5.6.7.8\noptions ndots:2\n",
		},
		{
			name:            "set all and keep host server",
			baseContent:     resolv2,
			nameservers:     []string{"1.2.3.4", "5.6.7.8"},
			options:         []string{"ndots:2"},
			searches:        []string{"test.com"},
			keepHostServers: true,
			want:            "search test.com example.com\nnameserver 1.2.3.4\nnameserver 5.6.7.8\nnameserver 1.1.1.1\noptions ndots:2 edns0\n",
		},
		{
			name:        "localhost nameservers should be filtered and use defaults instead",
			baseContent: "nameserver 127.0.0.1\nnameserver ::1\n",
			want:        "nameserver 8.8.8.8\nnameserver 8.8.4.4\n",
		},
		{
			name:        "localhost nameservers should not be filtered with hostns",
			baseContent: "nameserver 127.0.0.1\nnameserver ::1\n",
			hostns:      true,
			want:        "nameserver 127.0.0.1\nnameserver ::1\n",
		},
		{
			name:        "ipv6 nameservers should be filtered when ipv6 is not set",
			baseContent: "nameserver 1.1.1.1\nnameserver fd::1\n",
			want:        "nameserver 1.1.1.1\n",
		},
		{
			name:        "ipv6 nameservers should not be filtered when ipv6 is set",
			baseContent: "nameserver 1.1.1.1\nnameserver fd::1\n",
			ipv6:        true,
			want:        "nameserver 1.1.1.1\nnameserver fd::1\n",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			base := filepath.Join(t.TempDir(), "resolv.conf")
			target := filepath.Join(t.TempDir(), "new-resolv.conf")
			err := os.WriteFile(base, []byte(tt.baseContent), 0o644)
			assert.NoError(t, err, "write tmp resolv.conf")
			var namespaces []specs.LinuxNamespace
			if !tt.hostns {
				namespaces = []specs.LinuxNamespace{
					{Type: specs.NetworkNamespace},
				}
			}
			err = New(&Params{
				Path:            target,
				Nameservers:     tt.nameservers,
				Searches:        tt.searches,
				Options:         tt.options,
				IPv6Enabled:     tt.ipv6,
				KeepHostServers: tt.keepHostServers,
				Namespaces:      namespaces,
				resolvConfPath:  base,
			})
			assert.NoError(t, err, "New()")
			content, err := os.ReadFile(target)
			assert.NoError(t, err, "read tmp resolv.conf)")
			assert.Equal(t, tt.want, string(content), "expected resolv.conf does not match")
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		nameservers []string
		want        string
	}{
		{
			name:        "add single nameserver",
			content:     resolv1,
			nameservers: []string{"1.2.3.4"},
			want:        "nameserver 1.2.3.4\n" + resolv1,
		},
		{
			name:        "add single nameserver with search and options",
			content:     resolv2,
			nameservers: []string{"1.2.3.4"},
			want: `search example.com
nameserver 1.2.3.4
nameserver 1.1.1.1
options edns0
`,
		},
		{
			name:        "add three nameservers",
			content:     resolv1,
			nameservers: []string{"1.2.3.4", "2.3.4.5", "3.4.5.6"},
			want:        "nameserver 1.2.3.4\nnameserver 2.3.4.5\nnameserver 3.4.5.6\n" + resolv1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resolvPath := filepath.Join(t.TempDir(), "resolv.conf")
			err := os.WriteFile(resolvPath, []byte(tt.content), 0o644)
			assert.NoError(t, err, "write tmp resolv.conf")
			err = Add(resolvPath, tt.nameservers)
			assert.NoError(t, err, "Add()")
			content, err := os.ReadFile(resolvPath)
			assert.NoError(t, err, "read tmp resolv.conf)")
			assert.Equal(t, tt.want, string(content), "expected resolv.conf does not match")
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		nameservers []string
		want        string
	}{
		{
			name:        "remove single nameserver",
			content:     resolv1,
			nameservers: []string{"1.1.1.1"},
			want:        "",
		},
		{
			name:        "remove single nameserver with search and options",
			content:     resolv2,
			nameservers: []string{"1.1.1.1"},
			want: `search example.com
options edns0
`,
		},
		{
			name:        "remove three nameservers",
			content:     "nameserver 1.2.3.4\nnameserver 2.3.4.5\nnameserver 3.4.5.6\n" + resolv1,
			nameservers: []string{"1.2.3.4", "2.3.4.5", "3.4.5.6"},
			want:        resolv1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resolvPath := filepath.Join(t.TempDir(), "resolv.conf")
			err := os.WriteFile(resolvPath, []byte(tt.content), 0o644)
			assert.NoError(t, err, "write tmp resolv.conf")
			err = Remove(resolvPath, tt.nameservers)
			assert.NoError(t, err, "Remove()")
			content, err := os.ReadFile(resolvPath)
			assert.NoError(t, err, "read tmp resolv.conf)")
			assert.Equal(t, tt.want, string(content), "expected resolv.conf does not match")
		})
	}
}
