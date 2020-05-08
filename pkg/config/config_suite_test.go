package config

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
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
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(c).NotTo(gomega.BeNil())
	return c
}
