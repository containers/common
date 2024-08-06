package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultMachineVolumes(t *testing.T) {
	for _, tc := range []struct {
		envs   [][]string
		output []string
	}{
		{
			envs: [][]string{
				{"USERPROFILE", "C:\\Users\\test"},
			},
			output: []string{
				"C:\\Users\\test:/Users/test",
				"C:\\:/mnt/c",
			},
		},
		{
			envs: [][]string{
				{"USERPROFILE", "C:\\Users\\test\\AppData\\Local\\Temp\\podman_test123456789"},
			},
			output: []string{
				"C:\\Users\\test\\AppData\\Local\\Temp\\podman_test123456789:/Users/test/AppData/Local/Temp/podman_test123456789",
				"C:\\:/mnt/c",
			},
		},
		{
			envs: [][]string{
				{"USERPROFILE", "D:\\Users\\test"},
			},
			output: []string{
				"D:\\Users\\test:/Users/test",
				"D:\\:/mnt/d",
			},
		},
		{
			envs: [][]string{
				{"USERPROFILE", "c:\\users\\test"},
			},
			output: []string{
				"c:\\users\\test:/users/test",
				"c:\\:/mnt/c",
			},
		},
		{
			envs: [][]string{
				{"USERPROFILE", "C:/Users/test"},
			},
			output: []string{
				"C:/Users/test:/Users/test",
				"C:\\:/mnt/c",
			},
		},
		{
			envs: [][]string{
				{"USERPROFILE", "C:\\Users\\test\\"},
			},
			output: []string{
				"C:\\Users\\test\\:/Users/test/",
				"C:\\:/mnt/c",
			},
		},
	} {
		for _, env := range tc.envs {
			t.Setenv(env[0], env[1])
		}
		output := getDefaultMachineVolumes()
		assert.Equal(t, tc.output, output)
	}
}
