package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePodExitPolicy(t *testing.T) {
	tests := []struct {
		input    string
		expected PodExitPolicy
		mustFail bool
	}{
		{"", PodExitPolicyContinue, false},
		{"continue", PodExitPolicyContinue, false},
		{"stop", PodExitPolicyStop, false},
		{"-", PodExitPolicyUnsupported, true},
		{" stop", PodExitPolicyUnsupported, true},
		{"continue ", PodExitPolicyUnsupported, true},
		{"invalid", PodExitPolicyUnsupported, true},
	}

	for _, test := range tests {
		parsed, err := ParsePodExitPolicy(test.input)
		require.Equal(t, test.expected, parsed, "%v", test)
		if test.mustFail {
			require.Error(t, err, "%v", test)
		} else {
			require.NoError(t, err, "%v", test)
		}
	}
}
