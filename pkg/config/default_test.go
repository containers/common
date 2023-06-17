package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMachineVolumes(t *testing.T) {
	t.Parallel()

	os.Setenv("env1", "/test1")
	os.Setenv("env2", "/test2")
	for _, tc := range []struct {
		msg     string
		volumes []string
		output  []string
		assert  func(err error, in, out []string, msg string)
	}{
		{
			volumes: []string{},
			output:  []string{},
			assert: func(err error, in, out []string, msg string) {
				assert.Equal(t, in, out)
				assert.Nil(t, err, msg)
			},
		},
		{
			volumes: []string{"/foobar:/foobar", "/foobar1:/foobardest:ro", "$env1:/env1", "$env1:$env1", "$env2:$env1"},
			output:  []string{"/foobar:/foobar", "/foobar1:/foobardest:ro", "/test1:/env1", "/test1:/test1", "/test2:/test1"},
			assert: func(err error, in, out []string, msg string) {
				assert.Equal(t, in, out)
				assert.Nil(t, err, msg)
			},
		},
		{
			volumes: []string{"/foobar:/foobar", "/foobar1:/foobardest:ro", "$env1:/env1", "$env1:$env1", "$env3:$env1"},
			output:  []string{"/foobar:/foobar", "/foobar1:/foobardest:ro", "/test1:/env1", "/test1:/test1", "/test2:/test1"},
			assert: func(err error, in, out []string, msg string) {
				assert.EqualError(t, err, "invalid machine volume $env3:$env1, fields must container data", msg)
			},
		},
		{
			volumes: []string{"/foobar:/foobar", "/foobar1:/foobardest:ro", "$env1:/env1", "$env1:$env4", "$env1:$env1"},
			output:  []string{"/foobar:/foobar", "/foobar1:/foobardest:ro", "/test1:/env1", "/test1:/test1", "/test2:/test1"},
			assert: func(err error, in, out []string, msg string) {
				assert.EqualError(t, err, "invalid machine volume $env1:$env4, fields must container data", msg)
			},
		},
		{
			msg:     "This is broken",
			volumes: []string{"/foobar:/foobar:ro:badopt"},
			output:  []string{"/foobar:/foobar:ro"},
			assert: func(err error, in, out []string, msg string) {
				assert.EqualError(t, err, "invalid machine volume /foobar:/foobar:ro:badopt, 2 or 3 fields required", msg)
			},
		},
		{
			msg:     "This is broken2",
			volumes: []string{"/foobar"},
			output:  []string{"/foobar:/foobar:ro"},
			assert: func(err error, in, out []string, msg string) {
				assert.EqualError(t, err, "invalid machine volume /foobar, 2 or 3 fields required", msg)
			},
		},
	} {
		output, err := machineVolumes(tc.volumes)
		tc.assert(err, tc.output, output, tc.msg)
	}
}
