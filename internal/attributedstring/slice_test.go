package attributedstring

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Array Slice `toml:"array,omitempty"`
}

const (
	confingDefault    = `array=["1", "2", "3"]`
	configAppendFront = `array=[{append=true},"4", "5", "6"]`
	configAppendMid   = `array=["7", {append=true}, "8"]`
	configAppendBack  = `array=["9", {append=true}]`
	configAppendFalse = `array=["10", {append=false}]`
)

var (
	bTrue  = true
	bFalse = false
)

func loadConfigs(configs []string) (*testConfig, error) {
	var config testConfig
	for _, c := range configs {
		if _, err := toml.Decode(c, &config); err != nil {
			return nil, err
		}
	}
	return &config, nil
}

func TestSliceLoading(t *testing.T) {
	for _, test := range []struct {
		configs                []string
		expectedValues         []string
		expectedAppend         *bool
		expectedErrorSubstring string
	}{
		// Load single configs
		{[]string{confingDefault}, []string{"1", "2", "3"}, nil, ""},
		{[]string{configAppendFront}, []string{"4", "5", "6"}, &bTrue, ""},
		{[]string{configAppendMid}, []string{"7", "8"}, &bTrue, ""},
		{[]string{configAppendBack}, []string{"9"}, &bTrue, ""},
		{[]string{configAppendFalse}, []string{"10"}, &bFalse, ""},
		// Append=true
		{[]string{confingDefault, configAppendFront}, []string{"1", "2", "3", "4", "5", "6"}, &bTrue, ""},
		{[]string{configAppendFront, confingDefault}, []string{"4", "5", "6", "1", "2", "3"}, &bTrue, ""}, // The attribute is sticky unless explicitly being turned off in a later config
		{[]string{configAppendFront, confingDefault, configAppendBack}, []string{"4", "5", "6", "1", "2", "3", "9"}, &bTrue, ""},
		// Append=false
		{[]string{confingDefault, configAppendFalse}, []string{"10"}, &bFalse, ""},
		{[]string{confingDefault, configAppendMid, configAppendFalse}, []string{"10"}, &bFalse, ""},
		{[]string{confingDefault, configAppendFalse, configAppendMid}, []string{"10", "7", "8"}, &bTrue, ""}, // Append can be re-enabled by a later config

		// Error checks
		{[]string{`array=["1", false]`}, nil, nil, `unsupported item in attributed string slice: false`},
		{[]string{`array=["1", 42]`}, nil, nil, `unsupported item in attributed string slice: 42`}, // Stop a `int` such that it passes on 32bit as well
		{[]string{`array=["1", {foo=true}]`}, nil, nil, `unsupported key "foo" in map: `},
		{[]string{`array=["1", {append="false"}]`}, nil, nil, `unable to cast append to bool: `},
	} {
		result, err := loadConfigs(test.configs)
		if test.expectedErrorSubstring != "" {
			require.Error(t, err, "test is expected to fail: %v", test)
			require.ErrorContains(t, err, test.expectedErrorSubstring, "error does not match: %v", test)
			continue
		}
		require.NoError(t, err, "test is expected to succeed: %v", test)
		require.NotNil(t, result, "loaded config must not be nil: %v", test)
		require.Equal(t, result.Array.Values, test.expectedValues, "slices do not match: %v", test)
		require.Equal(t, result.Array.Attributes.Append, test.expectedAppend, "append field does not match: %v", test)
	}
}

func TestSliceEncoding(t *testing.T) {
	for _, test := range []struct {
		configs        []string
		marshalledData string
		expectedValues []string
		expectedAppend *bool
	}{
		{
			[]string{confingDefault},
			"array = [\"1\", \"2\", \"3\"]\n",
			[]string{"1", "2", "3"},
			nil,
		},
		{
			[]string{configAppendFront},
			"array = [\"4\", \"5\", \"6\", {append = true}]\n",
			[]string{"4", "5", "6"},
			&bTrue,
		},
		{
			[]string{configAppendFront, configAppendFalse},
			"array = [\"10\", {append = false}]\n",
			[]string{"10"},
			&bFalse,
		},
	} {
		// 1) Load the configs
		result, err := loadConfigs(test.configs)
		require.NoError(t, err, "loading config must succeed")
		require.NotNil(t, result, "loaded config must not be nil")
		require.Equal(t, result.Array.Values, test.expectedValues, "slices do not match: %v", test)
		require.Equal(t, result.Array.Attributes.Append, test.expectedAppend, "append field does not match: %v", test)

		// 2) Marshal the config to emulate writing it to disk
		buf := new(bytes.Buffer)
		enc := toml.NewEncoder(buf)
		encErr := enc.Encode(result)
		require.NoError(t, encErr, "encoding config must work")
		require.Equal(t, buf.String(), test.marshalledData)

		// 3) Reload the marshaled config to make sure that data is preserved
		var reloadedConfig testConfig
		_, decErr := toml.Decode(buf.String(), &reloadedConfig)
		require.NoError(t, decErr, "loading config must succeed")
		require.NotNil(t, reloadedConfig, "re-loaded config must not be nil")
		require.Equal(t, reloadedConfig.Array.Values, test.expectedValues, "slices do not match: %v", test)
		require.Equal(t, reloadedConfig.Array.Attributes.Append, test.expectedAppend, "append field does not match: %v", test)
	}
}
