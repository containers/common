package config_test

import (
	"testing"

	"github.com/containers/common/pkg/config"
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
	sut *config.Config
)

func beforeEach() {
	sut = defaultConfig()
}

func defaultConfig() *config.Config {
	c, err := config.DefaultConfig()
	Expect(err).To(BeNil())
	Expect(c).NotTo(BeNil())
	return c
}
