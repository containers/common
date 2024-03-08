package pasta

import "net"

const BinaryName = "pasta"

type SetupResult struct {
	// IpAddresses configured by pasta
	IPAddresses []net.IP
}
