//go:build (linux || freebsd) && cni

package cni_test

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/containers/common/libnetwork/types"
	"github.com/containers/common/libnetwork/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Config", func() {
	var (
		libpodNet  types.ContainerNetwork
		cniConfDir string
		logBuffer  bytes.Buffer
	)

	BeforeEach(func() {
		t := GinkgoT()
		cniConfDir = t.TempDir()
		logBuffer = bytes.Buffer{}
		logrus.SetOutput(&logBuffer)
		logrus.SetLevel(logrus.InfoLevel)
		DeferCleanup(func() {
			logrus.SetLevel(logrus.InfoLevel)
		})
	})

	JustBeforeEach(func() {
		var err error
		libpodNet, err = getNetworkInterface(cniConfDir)
		if err != nil {
			Fail("Failed to create NewCNINetworkInterface")
		}
	})

	Context("basic network config tests", func() {
		It("check default network config exists", func() {
			networks, err := libpodNet.NetworkList()
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(1))
			Expect(networks[0].Name).To(Equal("podman"))
			Expect(networks[0].Driver).To(Equal("bridge"))
			Expect(networks[0].NetworkInterface).To(Equal("cni-podman0"))
			Expect(networks[0].Created.Before(time.Now())).To(BeTrue())
			Expect(networks[0].Subnets).To(HaveLen(1))
			Expect(networks[0].Subnets[0].Subnet.String()).To(Equal("10.88.0.0/16"))
			Expect(networks[0].Subnets[0].Gateway.String()).To(Equal("10.88.0.1"))
			Expect(networks[0].Subnets[0].LeaseRange).To(BeNil())
			Expect(networks[0].IPAMOptions).To(HaveKeyWithValue("driver", "host-local"))
			Expect(networks[0].Options).To(BeEmpty())
			Expect(networks[0].Labels).To(BeEmpty())
			Expect(networks[0].DNSEnabled).To(BeFalse())
			Expect(networks[0].Internal).To(BeFalse())
		})

		It("basic network create, inspect and remove", func() {
			// Because we get the time from the file create timestamp there is small precision
			// loss so lets remove 500 milliseconds to make sure this test does not flake.
			now := time.Now().Add(-500 * time.Millisecond)
			network := types.Network{}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Labels).To(BeEmpty())
			Expect(network1.Options).To(BeEmpty())
			Expect(network1.IPAMOptions).ToNot(BeEmpty())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "host-local"))
			Expect(network1.Created.After(now)).To(BeTrue())
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal("10.89.0.0/24"))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.89.0.1"))
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.DNSEnabled).To(BeFalse())
			Expect(network1.Internal).To(BeFalse())

			// inspect by name
			network2, err := libpodNet.NetworkInspect(network1.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(network2).To(Equal(network1))

			// inspect by ID
			network2, err = libpodNet.NetworkInspect(network1.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(network2).To(Equal(network1))

			// inspect by partial ID
			network2, err = libpodNet.NetworkInspect(network1.ID[:10])
			Expect(err).ToNot(HaveOccurred())
			Expect(network2).To(Equal(network1))

			// create a new interface to force a config load from disk
			libpodNet, err = getNetworkInterface(cniConfDir)
			Expect(err).ToNot(HaveOccurred())

			network2, err = libpodNet.NetworkInspect(network1.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(network2).To(Equal(network1))

			err = libpodNet.NetworkRemove(network1.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(path).ToNot(BeARegularFile())

			_, err = libpodNet.NetworkInspect(network1.Name)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("network not found"))
		})

		It("create two networks", func() {
			// remove the dir here since we do not expect it to exists in the real context as well
			// the backend will create it for us
			os.RemoveAll(cniConfDir)

			network := types.Network{}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.Subnets).To(HaveLen(1))

			network = types.Network{}
			network2, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network2.Name).ToNot(Equal(network1.Name))
			Expect(network2.ID).ToNot(Equal(network1.ID))
			Expect(network2.NetworkInterface).ToNot(Equal(network1.NetworkInterface))
			Expect(network2.Subnets).To(HaveLen(1))
			Expect(network2.Subnets[0].Subnet.Contains(network1.Subnets[0].Subnet.IP)).To(BeFalse())
		})

		It("fail when creating two networks with the same name", func() {
			network := types.Network{}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.Subnets).To(HaveLen(1))

			network = types.Network{Name: network1.Name}
			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(MatchError(types.ErrNetworkExists))
		})

		It("return the same network when creating two networks with the same name and ignore", func() {
			network := types.Network{}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.Subnets).To(HaveLen(1))

			network = types.Network{Name: network1.Name}
			network2, err := libpodNet.NetworkCreate(network, &types.NetworkCreateOptions{IgnoreIfExists: true})
			Expect(err).ToNot(HaveOccurred())
			Expect(network2).To(Equal(network1))
		})

		It("create network with NetworDNSServers with DNSEnabled=false", func() {
			network := types.Network{
				NetworkDNSServers: []string{"8.8.8.8", "3.3.3.3"},
				DNSEnabled:        false,
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`NetworkDNSServers cannot be configured for backend CNI`))
		})

		It("create bridge config", func() {
			network := types.Network{Driver: "bridge"}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(filepath.Join(cniConfDir, network1.Name+".conflist")).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Labels).To(BeEmpty())
			Expect(network1.Options).To(BeEmpty())
			Expect(network1.IPAMOptions).ToNot(BeEmpty())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "host-local"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal("10.89.0.0/24"))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.89.0.1"))
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.DNSEnabled).To(BeFalse())
			Expect(network1.Internal).To(BeFalse())
		})

		It("create bridge config with com.docker.network.bridge.name", func() {
			network := types.Network{
				Driver: "bridge",
				Options: map[string]string{
					"com.docker.network.bridge.name": "foo",
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(filepath.Join(cniConfDir, network1.Name+".conflist")).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).To(Equal("foo"))
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Labels).To(BeEmpty())
			Expect(network1.Options).To(BeEmpty())
			Expect(network1.IPAMOptions).ToNot(BeEmpty())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "host-local"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal("10.89.0.0/24"))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.89.0.1"))
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.DNSEnabled).To(BeFalse())
			Expect(network1.Internal).To(BeFalse())
		})

		It("create bridge config with none ipam driver", func() {
			network := types.Network{
				Driver: "bridge",
				IPAMOptions: map[string]string{
					"driver": "none",
				},
				DNSEnabled: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.DNSEnabled).To(BeFalse())
			Expect(network1.IPAMOptions).ToNot(BeEmpty())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "none"))
			Expect(network1.Subnets).To(BeEmpty())

			// reload configs from disk
			libpodNet, err = getNetworkInterface(cniConfDir)
			Expect(err).ToNot(HaveOccurred())

			network2, err := libpodNet.NetworkInspect(network1.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(network2).To(Equal(network1))
		})

		It("create bridge config with none ipam driver and subnets", func() {
			subnet := "10.1.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			network := types.Network{
				Driver: "bridge",
				IPAMOptions: map[string]string{
					"driver": "none",
				},
				Subnets: []types.Subnet{
					{Subnet: n},
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("none ipam driver is set but subnets are given"))
		})

		It("create bridge config with dhcp ipam driver", func() {
			network := types.Network{
				Driver: "bridge",
				IPAMOptions: map[string]string{
					"driver": "dhcp",
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.IPAMOptions).ToNot(BeEmpty())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "dhcp"))
			Expect(network1.Subnets).To(BeEmpty())

			// reload configs from disk
			libpodNet, err = getNetworkInterface(cniConfDir)
			Expect(err).ToNot(HaveOccurred())

			network2, err := libpodNet.NetworkInspect(network1.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(network2).To(Equal(network1))
		})

		It("create bridge config with none hdcp driver and subnets", func() {
			subnet := "10.1.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			network := types.Network{
				Driver: "bridge",
				IPAMOptions: map[string]string{
					"driver": "dhcp",
				},
				Subnets: []types.Subnet{
					{Subnet: n},
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("dhcp ipam driver is set but subnets are given"))
		})

		It("create bridge with same name should fail", func() {
			network := types.Network{
				Driver:           "bridge",
				NetworkInterface: "cni-podman2",
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).To(Equal("cni-podman2"))
			Expect(network1.Driver).To(Equal("bridge"))

			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bridge name cni-podman2 already in use"))
		})

		It("create macvlan config", func() {
			network := types.Network{Driver: "macvlan"}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(filepath.Join(cniConfDir, network1.Name+".conflist")).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("macvlan"))
			Expect(network1.Labels).To(BeEmpty())
			Expect(network1.Options).To(BeEmpty())
			Expect(network1.IPAMOptions).ToNot(BeEmpty())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "dhcp"))
			Expect(network1.Subnets).To(BeEmpty())
			Expect(network1.DNSEnabled).To(BeFalse())
			Expect(network1.Internal).To(BeFalse())
		})

		It("create macvlan config with device", func() {
			network := types.Network{
				Driver:           "macvlan",
				NetworkInterface: "lo",
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("macvlan"))
			Expect(network1.Labels).To(BeEmpty())
			Expect(network1.Options).To(BeEmpty())
			Expect(network1.Subnets).To(BeEmpty())
			Expect(network1.DNSEnabled).To(BeFalse())
			Expect(network1.Internal).To(BeFalse())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "dhcp"))
			grepInFile(path, `"type": "macvlan"`)
			grepInFile(path, `"master": "lo"`)
			grepInFile(path, `"type": "dhcp"`)
		})

		It("create macvlan config with subnet", func() {
			subnet := "10.1.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			network := types.Network{
				Driver: "macvlan",
				Subnets: []types.Subnet{
					{Subnet: n},
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("macvlan"))
			Expect(network1.Labels).To(BeEmpty())
			Expect(network1.Options).To(BeEmpty())
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.1.0.1"))
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.DNSEnabled).To(BeFalse())
			Expect(network1.Internal).To(BeFalse())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "host-local"))
			grepInFile(path, `"type": "host-local"`)
		})

		// https://github.com/containers/podman/issues/12971
		It("create macvlan with a used subnet", func() {
			subnet := "127.0.0.0/8"
			n, _ := types.ParseCIDR(subnet)
			network := types.Network{
				Driver: "macvlan",
				Subnets: []types.Subnet{
					{Subnet: n},
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("macvlan"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("127.0.0.1"))
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "host-local"))
			grepInFile(path, `"type": "host-local"`)
		})

		It("create ipvlan config with subnet", func() {
			subnet := "10.1.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			network := types.Network{
				Driver: "ipvlan",
				Subnets: []types.Subnet{
					{Subnet: n},
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("ipvlan"))
			Expect(network1.Labels).To(BeEmpty())
			Expect(network1.Options).To(BeEmpty())
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.1.0.1"))
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.DNSEnabled).To(BeFalse())
			Expect(network1.Internal).To(BeFalse())
			Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "host-local"))
			grepInFile(path, `"type": "host-local"`)
		})

		It("create macvlan config with mode", func() {
			for _, mode := range []string{"bridge", "private", "vepa", "passthru"} {
				network := types.Network{
					Driver: "macvlan",
					Options: map[string]string{
						types.ModeOption: mode,
					},
				}
				network1, err := libpodNet.NetworkCreate(network, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(network1.Name).ToNot(BeEmpty())
				path := filepath.Join(cniConfDir, network1.Name+".conflist")
				Expect(path).To(BeARegularFile())
				Expect(network1.Driver).To(Equal("macvlan"))
				Expect(network1.Options).To(HaveKeyWithValue("mode", mode))
				Expect(network1.IPAMOptions).ToNot(BeEmpty())
				Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "dhcp"))
				grepInFile(path, `"mode": "`+mode+`"`)
			}
		})

		It("create macvlan config with invalid mode", func() {
			network := types.Network{
				Driver: "macvlan",
				Options: map[string]string{
					types.ModeOption: "test",
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`unknown macvlan mode "test"`))
		})

		It("create macvlan config with invalid device", func() {
			network := types.Network{
				Driver:           "macvlan",
				NetworkInterface: "idonotexists",
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parent interface idonotexists does not exist"))
		})

		It("create macvlan config with internal and dhcp should fail", func() {
			subnet := "10.1.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			network := types.Network{
				Driver:   "macvlan",
				Internal: true,
				Subnets: []types.Subnet{
					{Subnet: n},
				},
			}
			net1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(net1.Internal).To(BeTrue())
			path := filepath.Join(cniConfDir, net1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			grepNotFile(path, `"0.0.0.0/0"`)
		})

		It("create macvlan config with internal and subnet should not fail", func() {
			network := types.Network{
				Driver:   "macvlan",
				Internal: true,
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("internal is not supported with macvlan"))
		})

		for _, driver := range []string{"macvlan", "ipvlan"} {
			It(fmt.Sprintf("create %s config with none ipam driver", driver), func() {
				network := types.Network{
					Driver: driver,
					IPAMOptions: map[string]string{
						"driver": "none",
					},
				}
				network1, err := libpodNet.NetworkCreate(network, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(network1.Driver).To(Equal(driver))
				Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "none"))
				Expect(network1.Subnets).To(BeEmpty())

				// reload configs from disk
				libpodNet, err = getNetworkInterface(cniConfDir)
				Expect(err).ToNot(HaveOccurred())

				network2, err := libpodNet.NetworkInspect(network1.Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(network2).To(Equal(network1))
			})
		}

		It("create ipvlan config with mode", func() {
			for _, mode := range []string{"l2", "l3", "l3s"} {
				network := types.Network{
					Driver: "ipvlan",
					Options: map[string]string{
						types.ModeOption: mode,
					},
				}
				network1, err := libpodNet.NetworkCreate(network, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(network1.Name).ToNot(BeEmpty())
				path := filepath.Join(cniConfDir, network1.Name+".conflist")
				Expect(path).To(BeARegularFile())
				Expect(network1.Driver).To(Equal("ipvlan"))
				Expect(network1.Options).To(HaveKeyWithValue("mode", mode))
				Expect(network1.IPAMOptions).ToNot(BeEmpty())
				Expect(network1.IPAMOptions).To(HaveKeyWithValue("driver", "dhcp"))
				grepInFile(path, `"mode": "`+mode+`"`)

				// reload configs from disk
				libpodNet, err = getNetworkInterface(cniConfDir)
				Expect(err).ToNot(HaveOccurred())

				network2, err := libpodNet.NetworkInspect(network1.Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(network2).To(Equal(network1))
			}
		})

		It("create ipvlan config with invalid mode", func() {
			network := types.Network{
				Driver: "ipvlan",
				Options: map[string]string{
					types.ModeOption: "test",
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`unknown ipvlan mode "test"`))
		})

		It("create bridge with subnet", func() {
			subnet := "10.0.0.0/24"
			n, _ := types.ParseCIDR(subnet)

			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n},
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.0.0.1"))
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
		})

		It("create bridge with ipv6 subnet", func() {
			subnet := "fdcc::/64"
			n, _ := types.ParseCIDR(subnet)

			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n},
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.IPv6Enabled).To(BeTrue())
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("fdcc::1"))
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())

			// reload configs from disk
			libpodNet, err = getNetworkInterface(cniConfDir)
			Expect(err).ToNot(HaveOccurred())
			// check the networks are identical
			network2, err := libpodNet.NetworkInspect(network1.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1).To(Equal(network2))
		})

		It("create bridge with ipv6 enabled", func() {
			network := types.Network{
				Driver:      "bridge",
				IPv6Enabled: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(2))
			Expect(network1.Subnets[0].Subnet.String()).To(ContainSubstring(".0/24"))
			Expect(network1.Subnets[0].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.Subnets[1].Subnet.String()).To(ContainSubstring("::/64"))
			Expect(network1.Subnets[1].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[1].LeaseRange).To(BeNil())
		})

		It("create bridge with ipv6 enabled and ipv4 subnet", func() {
			subnet := "10.100.0.0/24"
			n, _ := types.ParseCIDR(subnet)

			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n},
				},
				IPv6Enabled: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(2))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.Subnets[1].Subnet.String()).To(ContainSubstring("::/64"))
			Expect(network1.Subnets[1].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[1].LeaseRange).To(BeNil())
		})

		It("create bridge with ipv6 enabled and ipv6 subnet", func() {
			subnet := "fd66::/64"
			n, _ := types.ParseCIDR(subnet)

			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n},
				},
				IPv6Enabled: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(2))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.Subnets[1].Subnet.String()).To(ContainSubstring(".0/24"))
			Expect(network1.Subnets[1].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[1].LeaseRange).To(BeNil())
		})

		It("create bridge with ipv6 enabled and ipv4+ipv6 subnet", func() {
			subnet1 := "10.100.0.0/24"
			n1, _ := types.ParseCIDR(subnet1)
			subnet2 := "fd66::/64"
			n2, _ := types.ParseCIDR(subnet2)

			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n1}, {Subnet: n2},
				},
				IPv6Enabled: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(2))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet1))
			Expect(network1.Subnets[0].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.Subnets[1].Subnet.String()).To(Equal(subnet2))
			Expect(network1.Subnets[1].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[1].LeaseRange).To(BeNil())
		})

		It("create bridge with ipv6 enabled and two ipv4 subnets", func() {
			subnet1 := "10.100.0.0/24"
			n1, _ := types.ParseCIDR(subnet1)
			subnet2 := "10.200.0.0/24"
			n2, _ := types.ParseCIDR(subnet2)

			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n1}, {Subnet: n2},
				},
				IPv6Enabled: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(3))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet1))
			Expect(network1.Subnets[0].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
			Expect(network1.Subnets[1].Subnet.String()).To(Equal(subnet2))
			Expect(network1.Subnets[1].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[1].LeaseRange).To(BeNil())
			Expect(network1.Subnets[2].Subnet.String()).To(ContainSubstring("::/64"))
			Expect(network1.Subnets[2].Gateway).ToNot(BeNil())
			Expect(network1.Subnets[2].LeaseRange).To(BeNil())
		})

		It("create bridge with subnet and gateway", func() {
			subnet := "10.0.0.5/24"
			n, _ := types.ParseCIDR(subnet)
			gateway := "10.0.0.50"
			g := net.ParseIP(gateway)
			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n, Gateway: g},
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal("10.0.0.0/24"))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal(gateway))
			Expect(network1.Subnets[0].LeaseRange).To(BeNil())
		})

		It("create bridge with subnet and gateway not in the same subnet", func() {
			subnet := "10.0.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			gateway := "10.10.0.50"
			g := net.ParseIP(gateway)
			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n, Gateway: g},
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not in subnet"))
		})

		It("create bridge with subnet and lease range", func() {
			subnet := "10.0.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			startIP := "10.0.0.10"
			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n, LeaseRange: &types.LeaseRange{
						StartIP: net.ParseIP(startIP),
					}},
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.0.0.1"))
			Expect(network1.Subnets[0].LeaseRange.StartIP.String()).To(Equal(startIP))

			err = libpodNet.NetworkRemove(network1.Name)
			Expect(err).ToNot(HaveOccurred())

			endIP := "10.0.0.30"
			network = types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n, LeaseRange: &types.LeaseRange{
						EndIP: net.ParseIP(endIP),
					}},
				},
			}
			network1, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(filepath.Join(cniConfDir, network1.Name+".conflist")).To(BeARegularFile())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.0.0.1"))
			Expect(network1.Subnets[0].LeaseRange.EndIP.String()).To(Equal(endIP))

			err = libpodNet.NetworkRemove(network1.Name)
			Expect(err).ToNot(HaveOccurred())

			network = types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n, LeaseRange: &types.LeaseRange{
						StartIP: net.ParseIP(startIP),
						EndIP:   net.ParseIP(endIP),
					}},
				},
			}
			network1, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.0.0.1"))
			Expect(network1.Subnets[0].LeaseRange.StartIP.String()).To(Equal(startIP))
			Expect(network1.Subnets[0].LeaseRange.EndIP.String()).To(Equal(endIP))

			// create a new interface to force a config load from disk
			libpodNet, err = getNetworkInterface(cniConfDir)
			Expect(err).ToNot(HaveOccurred())

			network1, err = libpodNet.NetworkInspect(network1.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(BeEmpty())
			Expect(network1.ID).ToNot(BeEmpty())
			Expect(network1.NetworkInterface).ToNot(BeEmpty())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).To(Equal(subnet))
			Expect(network1.Subnets[0].Gateway.String()).To(Equal("10.0.0.1"))
			Expect(network1.Subnets[0].LeaseRange.StartIP.String()).To(Equal(startIP))
			Expect(network1.Subnets[0].LeaseRange.EndIP.String()).To(Equal(endIP))
		})

		It("create bridge with subnet and invalid lease range", func() {
			subnet := "10.0.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			startIP := "10.0.1.2"
			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n, LeaseRange: &types.LeaseRange{
						StartIP: net.ParseIP(startIP),
					}},
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not in subnet"))

			endIP := "10.1.1.1"
			network = types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: n, LeaseRange: &types.LeaseRange{
						EndIP: net.ParseIP(endIP),
					}},
				},
			}
			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not in subnet"))
		})

		It("create bridge with broken subnet", func() {
			network := types.Network{
				Driver: "bridge",
				Subnets: []types.Subnet{
					{Subnet: types.IPNet{}},
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("subnet ip is nil"))
		})

		It("create network with name", func() {
			name := "myname"
			network := types.Network{
				Name: name,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).To(Equal(name))
			Expect(network1.NetworkInterface).ToNot(Equal(name))
			Expect(network1.Driver).To(Equal("bridge"))
		})

		It("create network with invalid name", func() {
			name := "myname@some"
			network := types.Network{
				Name: name,
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
		})

		It("create network with name", func() {
			name := "myname"
			network := types.Network{
				Name: name,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).To(Equal(name))
			Expect(network1.NetworkInterface).ToNot(Equal(name))
			Expect(network1.Driver).To(Equal("bridge"))
		})

		It("create network with invalid name", func() {
			name := "myname@some"
			network := types.Network{
				Name: name,
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
		})

		It("create network with interface name", func() {
			name := "myname"
			network := types.Network{
				NetworkInterface: name,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).ToNot(Equal(name))
			Expect(network1.NetworkInterface).To(Equal(name))
			Expect(network1.Driver).To(Equal("bridge"))
		})

		It("create network with invalid interface name", func() {
			name := "myname@some"
			network := types.Network{
				NetworkInterface: name,
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
		})

		It("create bridge config with invalid com.docker.network.bridge.name", func() {
			network := types.Network{
				Driver: "bridge",
				Options: map[string]string{
					"com.docker.network.bridge.name": "myname@some",
				},
			}

			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
		})

		It("create network with ID should fail", func() {
			id := "17f29b073143d8cd97b5bbe492bdeffec1c5fee55cc1fe2112c8b9335f8b6121"
			network := types.Network{
				ID: id,
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ID can not be set for network create"))
		})

		It("create bridge with dns", func() {
			SkipIfNoDnsname()
			network := types.Network{
				Driver:     "bridge",
				DNSEnabled: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.DNSEnabled).To(BeTrue())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			grepInFile(path, `"type": "dnsname"`)
		})

		It("create bridge with internal", func() {
			network := types.Network{
				Driver:   "bridge",
				Internal: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).ToNot(BeEmpty())
			Expect(network1.Subnets[0].Gateway).To(BeNil())
			Expect(network1.Internal).To(BeTrue())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			grepNotFile(path, `"0.0.0.0/0"`)
		})

		It("create network with labels", func() {
			network := types.Network{
				Labels: map[string]string{
					"key": "value",
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Labels).ToNot(BeNil())
			Expect(network1.Labels).To(ContainElement("value"))
		})

		It("create network with mtu option", func() {
			network := types.Network{
				Options: map[string]string{
					types.MTUOption: "1500",
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Options).ToNot(BeNil())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			grepInFile(path, `"mtu": 1500,`)
			Expect(network1.Options).To(HaveKeyWithValue("mtu", "1500"))
		})

		It("create network with invalid mtu option", func() {
			network := types.Network{
				Options: map[string]string{
					types.MTUOption: "abc",
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`parsing "abc": invalid syntax`))

			network = types.Network{
				Options: map[string]string{
					types.MTUOption: "-1",
				},
			}
			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`mtu -1 is less than zero`))
		})

		It("create network with com.docker.network.driver.mtu option", func() {
			network := types.Network{
				Options: map[string]string{
					"com.docker.network.driver.mtu": "1500",
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Options).ToNot(BeNil())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			grepInFile(path, `"mtu": "1500"`)
			Expect(network1.Options).To(HaveKeyWithValue("mtu", "1500"))
			Expect(network1.Options).ToNot(HaveKeyWithValue("com.docker.network.driver.mtu", "1500"))
		})

		It("create network with invalid com.docker.network.driver.mtu option", func() {
			network := types.Network{
				Options: map[string]string{
					"com.docker.network.driver.mtu": "abc",
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`parsing "abc": invalid syntax`))

			network = types.Network{
				Options: map[string]string{
					"com.docker.network.driver.mtu": "-1",
				},
			}
			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`mtu -1 is less than zero`))
		})

		It("create macvlan network with mtu option", func() {
			network := types.Network{
				Driver: "macvlan",
				Options: map[string]string{
					types.MTUOption: "1500",
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("macvlan"))
			Expect(network1.Options).ToNot(BeNil())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			grepInFile(path, `"mtu": 1500`)
			Expect(network1.Options).To(HaveKeyWithValue("mtu", "1500"))
		})

		It("create network with vlan option", func() {
			network := types.Network{
				Options: map[string]string{
					types.VLANOption: "5",
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Options).ToNot(BeNil())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			grepInFile(path, `"vlan": 5,`)
			Expect(network1.Options).To(HaveKeyWithValue("vlan", "5"))
		})

		It("create network with invalid vlan option", func() {
			network := types.Network{
				Options: map[string]string{
					types.VLANOption: "abc",
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`parsing "abc": invalid syntax`))

			network = types.Network{
				Options: map[string]string{
					types.VLANOption: "-1",
				},
			}
			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`vlan ID -1 must be between 0 and 4094`))
		})

		It("network create unsupported option", func() {
			network := types.Network{Options: map[string]string{
				"someopt": "",
			}}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported network option someopt"))
		})

		It("network create unsupported driver", func() {
			network := types.Network{
				Driver: "someDriver",
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported driver someDriver"))
		})

		It("network create internal and dns", func() {
			SkipIfNoDnsname()
			network := types.Network{
				Driver:     "bridge",
				Internal:   true,
				DNSEnabled: true,
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Subnets).To(HaveLen(1))
			Expect(network1.Subnets[0].Subnet.String()).ToNot(BeEmpty())
			Expect(network1.Subnets[0].Gateway).To(BeNil())
			Expect(network1.Internal).To(BeTrue())
			// internal and dns does not work, dns should be disabled
			Expect(network1.DNSEnabled).To(BeFalse())
			logString := logBuffer.String()
			Expect(logString).To(ContainSubstring("dnsname and internal networks are incompatible"))
		})

		It("network inspect partial ID", func() {
			network := types.Network{Name: "net4"}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.ID).To(Equal("b44b7426c006839e7fe6f15d1faf64db58079d5233cba09b43be2257c1652cf5"))
			network = types.Network{Name: "net5"}
			network1, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.ID).To(Equal("b67e86fb039828ad686aa13667975b9e51f192eb617044faf06cded9d31602af"))

			// Note ID is the sha256 from the name
			// both net4 and net5 have an ID starting with b...
			_, err = libpodNet.NetworkInspect("b")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("more than one result for network ID"))
		})

		It("network create two with same name", func() {
			network := types.Network{Name: "net"}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Name).To(Equal("net"))
			network = types.Network{Name: "net"}
			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("network name net already used"))
		})

		It("remove default network config should fail", func() {
			err := libpodNet.NetworkRemove("podman")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("default network podman cannot be removed"))

			network, err := libpodNet.NetworkInspect("podman")
			Expect(err).ToNot(HaveOccurred())
			err = libpodNet.NetworkRemove(network.ID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("default network podman cannot be removed"))
		})

		It("network create with same subnet", func() {
			subnet := "10.0.0.0/24"
			n, _ := types.ParseCIDR(subnet)
			subnet2 := "10.10.0.0/24"
			n2, _ := types.ParseCIDR(subnet2)
			network := types.Network{Subnets: []types.Subnet{{Subnet: n}, {Subnet: n2}}}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Subnets).To(HaveLen(2))
			network = types.Network{Subnets: []types.Subnet{{Subnet: n}}}
			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("subnet 10.0.0.0/24 is already used on the host or by another config"))
			network = types.Network{Subnets: []types.Subnet{{Subnet: n2}}}
			_, err = libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("subnet 10.10.0.0/24 is already used on the host or by another config"))
		})

		It("create network with invalid ipam driver", func() {
			network := types.Network{
				Driver: "bridge",
				IPAMOptions: map[string]string{
					"driver": "blah",
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unsupported ipam driver \"blah\""))
		})

		It("create network with isolate option", func() {
			network := types.Network{
				Options: map[string]string{
					types.IsolateOption: "true",
				},
			}
			network1, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Driver).To(Equal("bridge"))
			Expect(network1.Options).ToNot(BeNil())
			path := filepath.Join(cniConfDir, network1.Name+".conflist")
			Expect(path).To(BeARegularFile())
			grepInFile(path, `"ingressPolicy": "same-bridge"`)
			Expect(network1.Options).To(HaveKeyWithValue("isolate", "true"))
			// reload configs from disk
			libpodNet, err = getNetworkInterface(cniConfDir)
			Expect(err).ToNot(HaveOccurred())

			network1, err = libpodNet.NetworkInspect(network1.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(network1.Options).To(HaveKeyWithValue("isolate", "true"))
		})

		It("create network with invalid isolate option", func() {
			network := types.Network{
				Options: map[string]string{
					types.IsolateOption: "123",
				},
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("network load valid existing ones", func() {
		numberOfConfigFiles := 0

		BeforeEach(func() {
			dir := "testfiles/valid"
			files, err := os.ReadDir(dir)
			if err != nil {
				Fail("Failed to read test directory")
			}
			for _, file := range files {
				filename := file.Name()
				data, err := os.ReadFile(filepath.Join(dir, filename))
				if err != nil {
					Fail("Failed to copy test files")
				}
				err = os.WriteFile(filepath.Join(cniConfDir, filename), data, 0o700)
				if err != nil {
					Fail("Failed to copy test files")
				}
			}
			numberOfConfigFiles = len(files)
		})

		It("load networks from disk", func() {
			logrus.SetLevel(logrus.WarnLevel)
			nets, err := libpodNet.NetworkList()
			Expect(err).ToNot(HaveOccurred())
			Expect(nets).To(HaveLen(numberOfConfigFiles))
			// test the we do not show logrus warnings/errors
			logString := logBuffer.String()
			Expect(logString).To(BeEmpty())
		})

		It("load networks from disk with log level debug", func() {
			logrus.SetLevel(logrus.DebugLevel)
			nets, err := libpodNet.NetworkList()
			Expect(err).ToNot(HaveOccurred())
			Expect(nets).To(HaveLen(numberOfConfigFiles))
			// check for the unsupported ipam plugin message
			logString := logBuffer.String()
			Expect(logString).ToNot(BeEmpty())
			Expect(logString).To(ContainSubstring("unsupported ipam plugin \\\"static\\\" in %s", cniConfDir+"/ipam-static.conflist"))
		})

		It("change network struct fields should not affect network struct in the backend", func() {
			nets, err := libpodNet.NetworkList()
			Expect(err).ToNot(HaveOccurred())
			Expect(nets).To(HaveLen(numberOfConfigFiles))

			nets[0].Name = "myname"
			nets, err = libpodNet.NetworkList()
			Expect(err).ToNot(HaveOccurred())
			Expect(nets).To(HaveLen(numberOfConfigFiles))
			Expect(nets).ToNot(ContainElement(HaveNetworkName("myname")))

			network, err := libpodNet.NetworkInspect("bridge")
			Expect(err).ToNot(HaveOccurred())
			network.NetworkInterface = "abc"

			network, err = libpodNet.NetworkInspect("bridge")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.NetworkInterface).ToNot(Equal("abc"))
		})

		It("bridge network", func() {
			network, err := libpodNet.NetworkInspect("bridge")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("bridge"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.NetworkInterface).To(Equal("cni-podman9"))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(HaveLen(1))
			Expect(network.Subnets[0].Subnet.String()).To(Equal("10.89.8.0/24"))
			Expect(network.Subnets[0].Gateway.String()).To(Equal("10.89.8.1"))
			Expect(network.Subnets[0].LeaseRange).ToNot(BeNil())
			Expect(network.Subnets[0].LeaseRange.StartIP.String()).To(Equal("10.89.8.20"))
			Expect(network.Subnets[0].LeaseRange.EndIP.String()).To(Equal("10.89.8.50"))
			Expect(network.Internal).To(BeFalse())
		})

		It("macvlan network", func() {
			network, err := libpodNet.NetworkInspect("macvlan")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("macvlan"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.NetworkInterface).To(Equal("lo"))
			Expect(network.Driver).To(Equal("macvlan"))
			Expect(network.Subnets).To(BeEmpty())
			// DHCP
			Expect(network.IPAMOptions).To(HaveKeyWithValue("driver", "dhcp"))
		})

		It("internal network", func() {
			network, err := libpodNet.NetworkInspect("internal")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("internal"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.NetworkInterface).To(Equal("cni-podman8"))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(HaveLen(1))
			Expect(network.Subnets[0].Subnet.String()).To(Equal("10.89.7.0/24"))
			Expect(network.Subnets[0].Gateway).To(BeNil())
			Expect(network.Internal).To(BeTrue())
		})

		It("bridge network with mtu", func() {
			network, err := libpodNet.NetworkInspect("mtu")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("mtu"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.NetworkInterface).To(Equal("cni-podman13"))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(HaveLen(1))
			Expect(network.Subnets[0].Subnet.String()).To(Equal("10.89.11.0/24"))
			Expect(network.Subnets[0].Gateway.String()).To(Equal("10.89.11.1"))
			Expect(network.Internal).To(BeFalse())
			Expect(network.Options).To(HaveLen(1))
			Expect(network.Options).To(HaveKeyWithValue("mtu", "1500"))
		})

		It("macvlan network with mtu", func() {
			network, err := libpodNet.NetworkInspect("macvlan_mtu")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("macvlan_mtu"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.NetworkInterface).To(Equal("lo"))
			Expect(network.Driver).To(Equal("macvlan"))
			Expect(network.Subnets).To(BeEmpty())
			Expect(network.Internal).To(BeFalse())
			Expect(network.Options).To(HaveLen(1))
			Expect(network.Options).To(HaveKeyWithValue("mtu", "1300"))
			Expect(network.IPAMOptions).To(HaveLen(1))
			Expect(network.IPAMOptions).To(HaveKeyWithValue("driver", "dhcp"))
		})

		It("bridge network with vlan", func() {
			network, err := libpodNet.NetworkInspect("vlan")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("vlan"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.NetworkInterface).To(Equal("cni-podman14"))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(HaveLen(1))
			Expect(network.Options).To(HaveLen(1))
			Expect(network.Options).To(HaveKeyWithValue("vlan", "5"))
		})

		It("bridge network with labels", func() {
			network, err := libpodNet.NetworkInspect("label")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("label"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.NetworkInterface).To(Equal("cni-podman15"))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(HaveLen(1))
			Expect(network.Labels).To(HaveLen(1))
			Expect(network.Labels).To(HaveKeyWithValue("mykey", "value"))
		})

		It("dual stack network", func() {
			network, err := libpodNet.NetworkInspect("dualstack")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("dualstack"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.NetworkInterface).To(Equal("cni-podman21"))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(HaveLen(2))

			sub1, _ := types.ParseCIDR("fd10:88:a::/64")
			sub2, _ := types.ParseCIDR("10.89.19.0/24")
			Expect(network.Subnets).To(ContainElements(
				types.Subnet{Subnet: sub1, Gateway: net.ParseIP("fd10:88:a::1")},
				types.Subnet{Subnet: sub2, Gateway: net.ParseIP("10.89.19.10").To4()},
			))
		})

		It("ipam static network", func() {
			network, err := libpodNet.NetworkInspect("ipam-static")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("ipam-static"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(BeEmpty())
			Expect(network.IPAMOptions).To(HaveKeyWithValue("driver", "static"))
		})

		It("ipam none network", func() {
			network, err := libpodNet.NetworkInspect("ipam-none")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("ipam-none"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(BeEmpty())
			Expect(network.IPAMOptions).To(HaveKeyWithValue("driver", "none"))
		})

		It("ipam empty network", func() {
			network, err := libpodNet.NetworkInspect("ipam-empty")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("ipam-empty"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Subnets).To(BeEmpty())
			Expect(network.IPAMOptions).To(HaveKeyWithValue("driver", "none"))
		})

		It("bridge with isolate option", func() {
			network, err := libpodNet.NetworkInspect("isolate")
			Expect(err).ToNot(HaveOccurred())
			Expect(network.Name).To(Equal("isolate"))
			Expect(network.ID).To(HaveLen(64))
			Expect(network.Driver).To(Equal("bridge"))
			Expect(network.Options).To(HaveKeyWithValue("isolate", "true"))
		})

		It("network list with filters (name)", func() {
			filters := map[string][]string{
				"name": {"internal", "bridge"},
			}
			filterFuncs, err := util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err := libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(2))
			Expect(networks).To(ConsistOf(HaveNetworkName("internal"), HaveNetworkName("bridge")))
		})

		It("network list with filters (partial name)", func() {
			filters := map[string][]string{
				"name": {"inte", "bri"},
			}
			filterFuncs, err := util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err := libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(2))
			Expect(networks).To(ConsistOf(HaveNetworkName("internal"), HaveNetworkName("bridge")))
		})

		It("network list with filters (id)", func() {
			filters := map[string][]string{
				"id": {"3bed2cb3a3acf7b6a8ef408420cc682d5520e26976d354254f528c965612054f", "17f29b073143d8cd97b5bbe492bdeffec1c5fee55cc1fe2112c8b9335f8b6121"},
			}
			filterFuncs, err := util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err := libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(2))
			Expect(networks).To(ConsistOf(HaveNetworkName("internal"), HaveNetworkName("bridge")))
		})

		It("network list with filters (id)", func() {
			filters := map[string][]string{
				"id": {"3bed2cb3a3acf7b6a8ef408420cc682d5520e26976d354254f528c965612054f", "17f29b073143d8cd97b5bbe492bdeffec1c5fee55cc1fe2112c8b9335f8b6121"},
			}
			filterFuncs, err := util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err := libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(2))
			Expect(networks).To(ConsistOf(HaveNetworkName("internal"), HaveNetworkName("bridge")))
		})

		It("network list with filters (partial id)", func() {
			filters := map[string][]string{
				"id": {"3bed2cb3a3acf7b6a8ef408420", "17f29b073143d8cd97b5bbe492bde"},
			}
			filterFuncs, err := util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err := libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(2))
			Expect(networks).To(ConsistOf(HaveNetworkName("internal"), HaveNetworkName("bridge")))
		})

		It("network list with filters (driver)", func() {
			filters := map[string][]string{
				"driver": {"bridge", "macvlan"},
			}
			filterFuncs, err := util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err := libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(numberOfConfigFiles))
			Expect(networks).To(ConsistOf(HaveNetworkName("internal"), HaveNetworkName("bridge"),
				HaveNetworkName("mtu"), HaveNetworkName("vlan"), HaveNetworkName("podman"),
				HaveNetworkName("label"), HaveNetworkName("macvlan"), HaveNetworkName("macvlan_mtu"),
				HaveNetworkName("dualstack"), HaveNetworkName("ipam-none"), HaveNetworkName("ipam-empty"),
				HaveNetworkName("ipam-static"), HaveNetworkName("isolate")))
		})

		It("network list with filters (label)", func() {
			filters := map[string][]string{
				"label": {"mykey"},
			}
			filterFuncs, err := util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err := libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(1))
			Expect(networks).To(ConsistOf(HaveNetworkName("label")))

			filters = map[string][]string{
				"label": {"mykey=value"},
			}
			filterFuncs, err = util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err = libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(1))
			Expect(networks).To(ConsistOf(HaveNetworkName("label")))
		})

		It("network list with filters", func() {
			filters := map[string][]string{
				"driver": {"bridge"},
				"label":  {"mykey"},
			}
			filterFuncs, err := util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())
			Expect(filterFuncs).To(HaveLen(2))

			networks, err := libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(HaveLen(1))
			Expect(networks).To(ConsistOf(HaveNetworkName("label")))

			filters = map[string][]string{
				"driver": {"macvlan"},
				"label":  {"mykey"},
			}
			filterFuncs, err = util.GenerateNetworkFilters(filters)
			Expect(err).ToNot(HaveOccurred())

			networks, err = libpodNet.NetworkList(filterFuncs...)
			Expect(err).ToNot(HaveOccurred())
			Expect(networks).To(BeEmpty())
		})

		It("create bridge network with used interface name", func() {
			network := types.Network{
				NetworkInterface: "cni-podman9",
			}
			_, err := libpodNet.NetworkCreate(network, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bridge name cni-podman9 already in use"))
		})
	})

	Context("network load invalid existing ones", func() {
		BeforeEach(func() {
			dir := "testfiles/invalid"
			files, err := os.ReadDir(dir)
			if err != nil {
				Fail("Failed to read test directory")
			}
			for _, file := range files {
				filename := file.Name()
				data, err := os.ReadFile(filepath.Join(dir, filename))
				if err != nil {
					Fail("Failed to copy test files")
				}
				err = os.WriteFile(filepath.Join(cniConfDir, filename), data, 0o700)
				if err != nil {
					Fail("Failed to copy test files")
				}
			}
		})

		It("load invalid networks from disk", func() {
			nets, err := libpodNet.NetworkList()
			Expect(err).ToNot(HaveOccurred())
			Expect(nets).To(HaveLen(2))
			logString := logBuffer.String()
			Expect(logString).To(ContainSubstring("noname.conflist: error parsing configuration list: no name"))
			Expect(logString).To(ContainSubstring("noplugin.conflist: error parsing configuration list: no plugins in list"))
			Expect(logString).To(ContainSubstring("invalidname.conflist has invalid name, skipping: names must match"))
			Expect(logString).To(ContainSubstring("has the same network name as"))
			Expect(logString).To(ContainSubstring("broken.conflist: error parsing configuration list"))
			Expect(logString).To(ContainSubstring("invalid_gateway.conflist could not be converted to a libpod config, skipping: failed to parse gateway ip 10.89.8"))
		})
	})
})

func grepInFile(path, match string) {
	grepFile(false, path, match)
}

func grepNotFile(path, match string) {
	grepFile(true, path, match)
}

func grepFile(not bool, path, match string) {
	data, err := os.ReadFile(path)
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	matcher := ContainSubstring(match)
	if not {
		matcher = Not(matcher)
	}
	ExpectWithOffset(2, string(data)).To(matcher)
}

// HaveNetworkName is a custom GomegaMatcher to match a network name
func HaveNetworkName(name string) gomegaTypes.GomegaMatcher {
	return WithTransform(func(e types.Network) string {
		return e.Name
	}, Equal(name))
}
