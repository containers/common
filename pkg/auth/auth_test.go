package auth

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
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
