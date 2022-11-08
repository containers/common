package report

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"json", true},
		{" json", true},
		{" json ", true},
		{"  json   ", true},
		// special case, previous regex allowed this template string but it is not actually a valid template
		{"{{json}}", true},
		{"{{json }}", true},
		{"{{json.}}", true},
		{"{{ json. }}", true},

		{"{{ json .}}", false},
		{"{{ json . }}", false},
		{"  {{   json   .  }}  ", false},
		{"{{ json .", false},
		{"json . }}", false},
		{"{{.ID }} json .", false},
		{"json .", false},
	}

	for _, tc := range tests {
		tc := tc
		label := "JSONFormat/" + strings.ReplaceAll(tc.input, " ", "_")
		t.Run(label, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, IsJSON(tc.input), fmt.Sprintf("Scanning %q failed", tc.input))
		})
	}
}
