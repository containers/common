package config

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

const (
	invalidPath = "/wrong"
)

var (
	sut *Config
)

func beforeEach() {
	sut = defaultConfig()
}

func defaultConfig() *Config {
	c, err := DefaultConfig()
	Expect(err).To(BeNil())
	Expect(c).NotTo(BeNil())
	return c
}
