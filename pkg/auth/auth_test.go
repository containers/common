package auth

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ = Describe("Config", func() {
	Describe("ValidateAuth", func() {
		It("validate GetDefaultAuthFile", func() {
			// Given
			oldDockerConf, envDockerSet := os.LookupEnv("DOCKER_CONFIG")
			os.Setenv("DOCKER_CONFIG", "/tmp")
			oldConf, envSet := os.LookupEnv("REGISTRY_AUTH_FILE")
			os.Setenv("REGISTRY_AUTH_FILE", "/tmp/registry.file")
			// When			// When
			authFile := GetDefaultAuthFile()
			// Then
			gomega.Expect(authFile).To(gomega.BeEquivalentTo("/tmp/registry.file"))
			os.Unsetenv("REGISTRY_AUTH_FILE")

			// Fall back to DOCKER_CONFIG
			authFile = GetDefaultAuthFile()
			// Then
			gomega.Expect(authFile).To(gomega.BeEquivalentTo("/tmp/config.json"))
			os.Unsetenv("DOCKER_CONFIG")

			// Fall back to DOCKER_CONFIG
			authFile = GetDefaultAuthFile()
			// Then
			gomega.Expect(authFile).To(gomega.BeEquivalentTo(""))

			// Undo that
			if envSet {
				os.Setenv("REGISTRY_AUTH_FILE", oldConf)
			}
			if envDockerSet {
				os.Setenv("DOCKER_CONFIG", oldDockerConf)
			}
		})
	})

	It("validate CheckAuthFile", func() {
		// When			// When
		err := CheckAuthFile("")
		// Then
		gomega.Expect(err).To(gomega.BeNil())

		conf, _ := ioutil.TempFile("", "authfile")
		defer os.Remove(conf.Name())
		// When			// When
		err = CheckAuthFile(conf.Name())
		// Then
		gomega.Expect(err).To(gomega.BeNil())

		// When			// When
		err = CheckAuthFile(conf.Name() + "missing")
		// Then
		gomega.Expect(err).ShouldNot(gomega.BeNil())
	})
})

func TestParseCredentialsKey(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		arg                string
		acceptRepositories bool
		expectedKey        string // or "" if we expect failure
		expectedRegistry   string
	}{
		{
			name:               "success repository",
			arg:                "quay.io/user",
			acceptRepositories: true,
			expectedKey:        "quay.io/user",
			expectedRegistry:   "quay.io",
		},
		{
			name:               "success no repository",
			arg:                "quay.io",
			acceptRepositories: true,
			expectedKey:        "quay.io",
			expectedRegistry:   "quay.io",
		},
		{
			name:               "a docker.io top-level namespace",
			arg:                "docker.io/user",
			acceptRepositories: true,
			expectedKey:        "docker.io/user",
			expectedRegistry:   "docker.io",
		},
		{
			name:               "a single docker.io/library repo",
			arg:                "docker.io/library/user",
			acceptRepositories: true,
			expectedKey:        "docker.io/library/user",
			expectedRegistry:   "docker.io",
		},
		{
			name:               "with http[s] prefix - accept repositories",
			arg:                "https://quay.io",
			acceptRepositories: true,
			expectedKey:        "quay.io",
			expectedRegistry:   "quay.io",
		},
		{
			name:               "with http[s] prefix - accept repositories",
			arg:                "http://example.com/v2/",
			acceptRepositories: true,
			expectedKey:        "example.com",
			expectedRegistry:   "example.com",
		},
		{
			name:               "with http[s] prefix + path - accept repositories",
			arg:                "https://quay.io/repo/imag:tag",
			acceptRepositories: true,
			expectedKey:        "quay.io",
			expectedRegistry:   "quay.io",
		},
		{
			name:               "with http[s] prefix + port - accept repositories",
			arg:                "https://quay.io:1234",
			acceptRepositories: true,
			expectedKey:        "quay.io:1234",
			expectedRegistry:   "quay.io:1234",
		},
		{
			name:               "with http[s] prefix + port + path - accept repositories",
			arg:                "https://quay.io:1234/repo@foo",
			acceptRepositories: true,
			expectedKey:        "quay.io:1234",
			expectedRegistry:   "quay.io:1234",
		},
		{
			name:               "with http[s] prefix",
			arg:                "https://quay.io",
			acceptRepositories: false,
			expectedKey:        "quay.io",
			expectedRegistry:   "quay.io",
		},
		{
			name:               "with http[s] prefix",
			arg:                "http://example.com/v2/",
			acceptRepositories: false,
			expectedKey:        "example.com",
			expectedRegistry:   "example.com",
		},
		{
			name:               "with http[s] prefix + path",
			arg:                "https://quay.io/repo/imag:tag",
			acceptRepositories: false,
			expectedKey:        "quay.io",
			expectedRegistry:   "quay.io",
		},
		{
			name:               "with http[s] prefix + port",
			arg:                "https://quay.io:1234",
			acceptRepositories: false,
			expectedKey:        "quay.io:1234",
			expectedRegistry:   "quay.io:1234",
		},
		{
			name:               "with http[s] prefix + port + path",
			arg:                "https://quay.io:1234/repo@foo",
			acceptRepositories: false,
			expectedKey:        "quay.io:1234",
			expectedRegistry:   "quay.io:1234",
		},
		{
			name:               "failure with tag",
			arg:                "quay.io/username/image:tag",
			acceptRepositories: true,
			expectedKey:        "",
		},
		{
			name:               "failure parse reference",
			arg:                "quay.io/:tag",
			acceptRepositories: true,
			expectedKey:        "",
		},
		{
			name:               "success accept no repository",
			arg:                "https://quay.io/user",
			acceptRepositories: false,
			expectedKey:        "quay.io",
			expectedRegistry:   "quay.io",
		},
	} {
		key, registry, err := parseCredentialsKey(tc.arg, tc.acceptRepositories)
		if tc.expectedKey == "" {
			assert.Error(t, err, tc.name)
		} else {
			require.NoError(t, err, tc.name)
			assert.Equal(t, tc.expectedKey, key, tc.name)
			assert.Equal(t, tc.expectedRegistry, registry)
		}
	}
}
