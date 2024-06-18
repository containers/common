package pasta

import (
	"testing"

	"github.com/containers/common/internal/attributedstring"
	"github.com/containers/common/libnetwork/types"
	"github.com/containers/common/pkg/config"
	"github.com/stretchr/testify/assert"
)

func makeSetupOptions(configArgs, extraArgs []string, ports []types.PortMapping) *SetupOptions {
	return &SetupOptions{
		Config:       &config.Config{Network: config.NetworkConfig{PastaOptions: attributedstring.NewSlice(configArgs)}},
		Netns:        "netns123",
		ExtraOptions: extraArgs,
		Ports:        ports,
	}
}

func Test_createPastaArgs(t *testing.T) {
	tests := []struct {
		name           string
		input          *SetupOptions
		wantArgs       []string
		wantDnsForward []string
		wantErr        string
	}{
		{
			name: "default options",
			input: makeSetupOptions(
				nil,
				nil,
				nil,
			),
			wantArgs: []string{
				"--config-net", "--dns-forward", dnsForwardIpv4, "-t", "none", "-u", "none",
				"-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "basic port",
			input: makeSetupOptions(
				nil,
				nil,
				[]types.PortMapping{{HostPort: 80, ContainerPort: 80, Protocol: "tcp", Range: 1}},
			),
			wantArgs: []string{
				"--config-net", "-t", "80-80:80-80", "--dns-forward", dnsForwardIpv4, "-u", "none",
				"-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "port range",
			input: makeSetupOptions(
				nil,
				nil,
				[]types.PortMapping{{HostPort: 80, ContainerPort: 80, Protocol: "tcp", Range: 3}},
			),
			wantArgs: []string{
				"--config-net", "-t", "80-82:80-82", "--dns-forward", dnsForwardIpv4, "-u", "none",
				"-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "different host and container port",
			input: makeSetupOptions(
				nil,
				nil,
				[]types.PortMapping{{HostPort: 80, ContainerPort: 60, Protocol: "tcp", Range: 1}},
			),
			wantArgs: []string{
				"--config-net", "-t", "80-80:60-60", "--dns-forward", dnsForwardIpv4, "-u", "none",
				"-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "tcp and udp port",
			input: makeSetupOptions(
				nil,
				nil,
				[]types.PortMapping{
					{HostPort: 80, ContainerPort: 60, Protocol: "tcp", Range: 1},
					{HostPort: 100, ContainerPort: 100, Protocol: "udp", Range: 1},
				},
			),
			wantArgs: []string{
				"--config-net", "-t", "80-80:60-60", "-u", "100-100:100-100", "--dns-forward",
				dnsForwardIpv4, "-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "two tcp ports",
			input: makeSetupOptions(
				nil,
				nil,
				[]types.PortMapping{
					{HostPort: 80, ContainerPort: 60, Protocol: "tcp", Range: 1},
					{HostPort: 100, ContainerPort: 100, Protocol: "tcp", Range: 1},
				},
			),
			wantArgs: []string{
				"--config-net", "-t", "80-80:60-60", "-t", "100-100:100-100", "--dns-forward",
				dnsForwardIpv4, "-u", "none", "-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "invalid port",
			input: makeSetupOptions(
				nil,
				nil,
				[]types.PortMapping{
					{HostPort: 80, ContainerPort: 60, Protocol: "sctp", Range: 1},
				},
			),
			wantErr: "can't forward protocol: sctp",
		},
		{
			name: "config options before extra options",
			input: makeSetupOptions(
				[]string{"-i", "eth0"},
				[]string{"-n", "24"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "-i", "eth0", "-n", "24", "--dns-forward", dnsForwardIpv4,
				"-t", "none", "-u", "none", "-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "config options before extra options",
			input: makeSetupOptions(
				[]string{"-i", "eth0"},
				[]string{"-n", "24"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "-i", "eth0", "-n", "24", "--dns-forward", dnsForwardIpv4,
				"-t", "none", "-u", "none", "-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "-T option",
			input: makeSetupOptions(
				nil,
				[]string{"-T", "80"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "-T", "80", "--dns-forward", dnsForwardIpv4,
				"-t", "none", "-u", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "--tcp-ns option",
			input: makeSetupOptions(
				nil,
				[]string{"--tcp-ns", "80"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "--tcp-ns", "80", "--dns-forward", dnsForwardIpv4,
				"-t", "none", "-u", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "--map-gw option",
			input: makeSetupOptions(
				nil,
				[]string{"--map-gw"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "--dns-forward", dnsForwardIpv4, "-t", "none",
				"-u", "none", "-T", "none", "-U", "none", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			// https://github.com/containers/podman/issues/22477
			name: "--map-gw with port directly after",
			input: makeSetupOptions(nil,
				[]string{"--map-gw", "-T", "80"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "-T", "80", "--dns-forward", dnsForwardIpv4,
				"-t", "none", "-u", "none", "-U", "none", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "two --map-gw",
			input: makeSetupOptions(
				[]string{"--map-gw", "-T", "80"},
				[]string{"--map-gw"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "-T", "80", "--dns-forward", dnsForwardIpv4,
				"-t", "none", "-u", "none", "-U", "none", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "--dns-forward option",
			input: makeSetupOptions(
				nil,
				[]string{"--dns-forward", "192.168.255.255"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "--dns-forward", "192.168.255.255", "-t", "none",
				"-u", "none", "-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{"192.168.255.255"},
		},
		{
			name: "two --dns-forward options",
			input: makeSetupOptions(
				nil,
				[]string{"--dns-forward", "192.168.255.255", "--dns-forward", "::1"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "--dns-forward", "192.168.255.255", "--dns-forward", "::1", "-t", "none",
				"-u", "none", "-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{"192.168.255.255", "::1"},
		},
		{
			name: "port and custom opt",
			input: makeSetupOptions(
				nil,
				[]string{"-i", "eth0"},
				[]types.PortMapping{{HostPort: 80, ContainerPort: 80, Protocol: "tcp", Range: 1}},
			),
			wantArgs: []string{
				"--config-net", "-i", "eth0", "-t", "80-80:80-80", "--dns-forward", dnsForwardIpv4,
				"-u", "none", "-T", "none", "-U", "none", "--no-map-gw", "--quiet", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
		{
			name: "Add verbose logging",
			input: makeSetupOptions(
				nil,
				[]string{"--log-file=/tmp/log", "--trace", "--debug"},
				nil,
			),
			wantArgs: []string{
				"--config-net", "--log-file=/tmp/log", "--trace", "--debug",
				"--dns-forward", dnsForwardIpv4, "-t", "none", "-u", "none", "-T", "none", "-U", "none",
				"--no-map-gw", "--netns", "netns123",
			},
			wantDnsForward: []string{dnsForwardIpv4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, dnsForward, err := createPastaArgs(tt.input)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr, "createPastaArgs error")
				return
			}
			assert.NoError(t, err, "expect no createPastaArgs error")
			assert.Equal(t, tt.wantArgs, args, "check arguments")
			assert.Equal(t, tt.wantDnsForward, dnsForward, "check dns forward")
		})
	}
}
