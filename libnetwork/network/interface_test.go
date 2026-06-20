//go:build linux || freebsd

package network

import "testing"

func TestRootlessModeForUser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		isRootless  bool
		rootlessUID int
		expect      bool
	}{
		{
			name:        "rootless user",
			isRootless:  true,
			rootlessUID: 1000,
			expect:      true,
		},
		{
			name:        "root in nested user namespace",
			isRootless:  true,
			rootlessUID: 0,
			expect:      false,
		},
		{
			name:        "rootful",
			isRootless:  false,
			rootlessUID: 0,
			expect:      false,
		},
	}

	for _, tcl := range testCases {
		tc := tcl
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := rootlessModeForUser(tc.isRootless, tc.rootlessUID); got != tc.expect {
				t.Fatalf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}
