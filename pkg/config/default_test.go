package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareProbeBinary(t *testing.T, expectedOutput string) (path string) {
	f, err := ioutil.TempFile("", "conmon-")
	require.Nil(t, err)
	defer func() { require.Nil(t, f.Close()) }()

	f.Chmod(os.FileMode(0755))
	f.WriteString(fmt.Sprintf("#!/usr/bin/env sh\necho %q", expectedOutput))
	return f.Name()
}

func TestProbeConmon(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		msg    string
		output string
		assert func(error, string)
	}{
		{
			msg:    "success conmon 2",
			output: "conmon version 2.1.0",
			assert: func(err error, msg string) {
				assert.Nil(t, err, msg)
			},
		},
		{
			msg:    "success conmon 3",
			output: "conmon version 3.0.0-dev",
			assert: func(x error, msg string) {
				assert.Nil(t, x, msg)
			},
		},
		{
			msg:    "failure outdated version",
			output: "conmon version 1.0.0",
			assert: func(err error, msg string) {
				assert.Equal(t, err, ErrConmonOutdated, msg)
			},
		},
		{
			msg:    "failure invalid format",
			output: "invalid",
			assert: func(err error, msg string) {
				assert.EqualError(t, err, _conmonVersionFormatErr, msg)
			},
		},
	} {
		filePath := prepareProbeBinary(t, tc.output)
		defer os.RemoveAll(filePath)
		err := probeConmon(filePath)
		tc.assert(err, tc.msg)
	}
}
