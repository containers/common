package auth

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/containers/image/v5/docker/reference"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
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

func TestParseRegistryArgument(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		arg                string
		acceptRepositories bool
		shouldErr          bool
		expect             func(key, registry string, ref reference.Named)
	}{
		{
			name:               "success repository",
			arg:                "quay.io/user",
			acceptRepositories: true,
			shouldErr:          false,
			expect: func(key, registry string, ref reference.Named) {
				assert.Equal(t, "quay.io/user", key)
				assert.Equal(t, "quay.io", registry)
				assert.NotNil(t, ref)
			},
		},
		{
			name:               "success no repository",
			arg:                "quay.io",
			acceptRepositories: true,
			shouldErr:          false,
			expect: func(key, registry string, ref reference.Named) {
				assert.Equal(t, "quay.io", key)
				assert.Equal(t, "quay.io", registry)
				assert.Nil(t, ref)
			},
		},
		{
			name:               "with http[s] prefix",
			arg:                "https://quay.io",
			acceptRepositories: true,
			shouldErr:          true,
		},
		{
			name:               "failure with tag",
			arg:                "quay.io/username/image:tag",
			acceptRepositories: true,
			shouldErr:          true,
		},
		{
			name:               "failure parse reference",
			arg:                "quay.io/:tag",
			acceptRepositories: true,
			shouldErr:          true,
		},
		{
			name:               "success accept no repository",
			arg:                "https://quay.io/user",
			acceptRepositories: false,
			shouldErr:          false,
			expect: func(key, registry string, ref reference.Named) {
				assert.Equal(t, "quay.io", key)
				assert.Equal(t, "quay.io", registry)
				assert.Nil(t, ref)
			},
		},
	} {
		key, registry, ref, err := parseRegistryArgument(tc.arg, tc.acceptRepositories)
		if tc.shouldErr {
			assert.Error(t, err, tc.name)
		} else {
			assert.NoError(t, err, tc.name)
			tc.expect(key, registry, ref)
		}
	}
}
