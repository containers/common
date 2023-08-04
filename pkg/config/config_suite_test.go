package config

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	gomega.RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

const (
	invalidPath = "/wrong"
)

var sut *Config

func beforeEach() {
	c, err := defaultConfig()
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(c).NotTo(gomega.BeNil())
	sut = c
}
