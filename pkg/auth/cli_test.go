package auth

import (
	"testing"

	"github.com/containers/common/pkg/completion"
	"github.com/spf13/pflag"
)

func testFlagCompletion(t *testing.T, flags *pflag.FlagSet, flagCompletions completion.FlagCompletions) {
	// lookup if for each flag a flag completion function exists
	flags.VisitAll(func(f *pflag.Flag) {
		// skip hidden, deprecated and boolean flags
		if f.Hidden || len(f.Deprecated) > 0 || f.Value.Type() == "bool" {
			return
		}
		if _, ok := flagCompletions[f.Name]; !ok {
			t.Errorf("Flag %q has no shell completion function set.", f.Name)
		}
	})

	// make sure no unnecessary flag completion functions are defined
	for name := range flagCompletions {
		if flag := flags.Lookup(name); flag == nil {
			t.Errorf("Flag %q does not exists but has a shell completion function set.", name)
		}
	}
}

func TestLoginFlagsCompletion(t *testing.T) {
	flags := GetLoginFlags(&LoginOptions{})
	flagCompletions := GetLoginFlagsCompletions()
	testFlagCompletion(t, flags, flagCompletions)
}

func TestLogoutFlagsCompletion(t *testing.T) {
	flags := GetLogoutFlags(&LogoutOptions{})
	flagCompletions := GetLogoutFlagsCompletions()
	testFlagCompletion(t, flags, flagCompletions)
}
