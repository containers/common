package flag

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionalBoolSet(t *testing.T) {
	for _, c := range []struct {
		input    string
		accepted bool
		value    bool
	}{
		// Valid inputs documented for strconv.ParseBool == flag.BoolVar
		{"1", true, true},
		{"t", true, true},
		{"T", true, true},
		{"TRUE", true, true},
		{"true", true, true},
		{"True", true, true},
		{"0", true, false},
		{"f", true, false},
		{"F", true, false},
		{"FALSE", true, false},
		{"false", true, false},
		{"False", true, false},
		// A few invalid inputs
		{"", false, false},
		{"yes", false, false},
		{"no", false, false},
		{"2", false, false},
	} {
		var ob OptionalBool
		v := internalNewOptionalBoolValue(&ob)
		require.False(t, ob.Present())
		err := v.Set(c.input)
		if c.accepted {
			assert.NoError(t, err, c.input)
			assert.Equal(t, c.value, ob.Value())
		} else {
			assert.Error(t, err, c.input)
			assert.False(t, ob.Present()) // Just to be extra paranoid.
		}
	}

	// Nothing actually explicitly says that .Set() is never called when the flag is not present on the command line;
	// so, check that it is not being called, at least in the straightforward case (it's not possible to test that it
	// is not called in any possible situation).
	var globalOB, commandOB OptionalBool
	actionRun := false
	app := &cobra.Command{
		Use: "app",
	}
	OptionalBoolFlag(app.PersistentFlags(), &globalOB, "global-OB", "")
	cmd := &cobra.Command{
		Use: "cmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			assert.False(t, globalOB.Present())
			assert.False(t, commandOB.Present())
			actionRun = true
			return nil
		},
	}
	OptionalBoolFlag(cmd.Flags(), &commandOB, "command-OB", "")
	app.AddCommand(cmd)
	app.SetArgs([]string{"cmd"})
	err := app.Execute()
	require.NoError(t, err)
	assert.True(t, actionRun)
}

func TestOptionalBoolString(t *testing.T) {
	for _, c := range []struct {
		input    OptionalBool
		expected string
	}{
		{OptionalBool{present: true, value: true}, "true"},
		{OptionalBool{present: true, value: false}, "false"},
		{OptionalBool{present: false, value: true}, ""},
		{OptionalBool{present: false, value: false}, ""},
	} {
		var ob OptionalBool
		v := internalNewOptionalBoolValue(&ob)
		ob = c.input
		res := v.String()
		assert.Equal(t, c.expected, res)
	}
}

func TestOptionalBoolIsBoolFlag(t *testing.T) {
	// IsBoolFlag means that the argument value must either be part of the same argument, with =;
	// if there is no =, the value is set to true.
	// This differs form other flags, where the argument is required and may be either separated with = or supplied in the next argument.
	for _, c := range []struct {
		input        []string
		expectedOB   OptionalBool
		expectedArgs []string
	}{
		{[]string{"1", "2"}, OptionalBool{present: false}, []string{"1", "2"}},                                       // Flag not present
		{[]string{"--OB=true", "1", "2"}, OptionalBool{present: true, value: true}, []string{"1", "2"}},              // --OB=true
		{[]string{"--OB=false", "1", "2"}, OptionalBool{present: true, value: false}, []string{"1", "2"}},            // --OB=false
		{[]string{"--OB", "true", "1", "2"}, OptionalBool{present: true, value: true}, []string{"true", "1", "2"}},   // --OB true
		{[]string{"--OB", "false", "1", "2"}, OptionalBool{present: true, value: true}, []string{"false", "1", "2"}}, // --OB false
	} {
		var ob OptionalBool
		actionRun := false
		app := &cobra.Command{Use: "app"}
		cmd := &cobra.Command{
			Use: "cmd",
			RunE: func(cmd *cobra.Command, args []string) error {
				assert.Equal(t, c.expectedOB, ob)     // nolint
				assert.Equal(t, c.expectedArgs, args) //nolint
				actionRun = true
				return nil
			},
		}
		OptionalBoolFlag(cmd.Flags(), &ob, "OB", "")
		app.AddCommand(cmd)

		app.SetArgs(append([]string{"cmd"}, c.input...))
		err := app.Execute()
		require.NoError(t, err)
		assert.True(t, actionRun)
	}
}

func TestOptionalStringSet(t *testing.T) {
	// Really just a smoke test, but differentiating between not present and empty.
	for _, c := range []string{"", "hello"} {
		var os OptionalString
		v := NewOptionalStringValue(&os)
		require.False(t, os.Present())
		err := v.Set(c)
		assert.NoError(t, err, c)
		assert.Equal(t, c, os.Value())
	}

	// Nothing actually explicitly says that .Set() is never called when the flag is not present on the command line;
	// so, check that it is not being called, at least in the straightforward case (it's not possible to test that it
	// is not called in any possible situation).
	var globalOS, commandOS OptionalString
	actionRun := false
	app := &cobra.Command{
		Use: "app",
	}
	app.PersistentFlags().Var(NewOptionalStringValue(&globalOS), "global-OS", "")
	cmd := &cobra.Command{
		Use: "cmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			assert.False(t, globalOS.Present())
			assert.False(t, commandOS.Present())
			actionRun = true
			return nil
		},
	}
	cmd.Flags().Var(NewOptionalStringValue(&commandOS), "command-OS", "")
	app.AddCommand(cmd)
	app.SetArgs([]string{"cmd"})
	err := app.Execute()
	require.NoError(t, err)
	assert.True(t, actionRun)
}

func TestOptionalStringString(t *testing.T) {
	for _, c := range []struct {
		input    OptionalString
		expected string
	}{
		{OptionalString{present: true, value: "hello"}, "hello"},
		{OptionalString{present: true, value: ""}, ""},
		{OptionalString{present: false, value: "hello"}, ""},
		{OptionalString{present: false, value: ""}, ""},
	} {
		var os OptionalString
		v := NewOptionalStringValue(&os)
		os = c.input
		res := v.String()
		assert.Equal(t, c.expected, res)
	}
}

func TestOptionalStringIsBoolFlag(t *testing.T) {
	// NOTE: OptionalStringValue does not implement IsBoolFlag!
	// IsBoolFlag means that the argument value must either be part of the same argument, with =;
	// if there is no =, the value is set to true.
	// This differs form other flags, where the argument is required and may be either separated with = or supplied in the next argument.
	for _, c := range []struct {
		input        []string
		expectedOS   OptionalString
		expectedArgs []string
	}{
		{[]string{"1", "2"}, OptionalString{present: false}, []string{"1", "2"}},                                 // Flag not present
		{[]string{"--OS=hello", "1", "2"}, OptionalString{present: true, value: "hello"}, []string{"1", "2"}},    // --OS=true
		{[]string{"--OS=", "1", "2"}, OptionalString{present: true, value: ""}, []string{"1", "2"}},              // --OS=false
		{[]string{"--OS", "hello", "1", "2"}, OptionalString{present: true, value: "hello"}, []string{"1", "2"}}, // --OS true
		{[]string{"--OS", "", "1", "2"}, OptionalString{present: true, value: ""}, []string{"1", "2"}},           // --OS false
	} {
		var os OptionalString
		actionRun := false
		app := &cobra.Command{
			Use: "app",
		}
		cmd := &cobra.Command{
			Use: "cmd",
			RunE: func(cmd *cobra.Command, args []string) error {
				assert.Equal(t, c.expectedOS, os)     // nolint
				assert.Equal(t, c.expectedArgs, args) // nolint
				actionRun = true
				return nil
			},
		}
		cmd.Flags().Var(NewOptionalStringValue(&os), "OS", "")
		app.AddCommand(cmd)
		app.SetArgs(append([]string{"cmd"}, c.input...))
		err := app.Execute()
		require.NoError(t, err)
		assert.True(t, actionRun)
	}
}
