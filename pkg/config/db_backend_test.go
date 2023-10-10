package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDBBackend(t *testing.T) {
	tests := []struct {
		input    string
		valid    bool
		expected DBBackend
	}{
		{stringBoltDB, true, DBBackendBoltDB},
		{stringSQLite, true, DBBackendSQLite},
		{"", true, DBBackendDefault},
		{stringSQLite + " ", false, DBBackendUnsupported},
	}

	for _, test := range tests {
		result, err := ParseDBBackend(test.input)
		if test.valid {
			require.NoError(t, err, "should parse %v", test)
			require.NoError(t, result.Validate(), "should validate %v", test)
			require.Equal(t, test.expected, result)
		} else {
			require.Error(t, err, "should NOT parse %v", test)
			require.Error(t, result.Validate(), "should NOT validate %v", test)
			require.Equal(t, test.expected, result)
		}
	}
}
