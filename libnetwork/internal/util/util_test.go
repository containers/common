package util

import (
	"net"
	"reflect"
	"testing"

	"github.com/containers/common/libnetwork/types"
	"github.com/containers/common/pkg/config"
)

func parseIPNet(subnet string) *types.IPNet {
	n, _ := types.ParseCIDR(subnet)
	return &n
}

func parseSubnet(subnet string) *types.Subnet {
	n := parseIPNet(subnet)
	return &types.Subnet{Subnet: *n}
}

func TestGetFreeIPv4NetworkSubnet(t *testing.T) {
	type args struct {
		usedNetworks []*net.IPNet
		subnetPools  []config.SubnetPool
	}
	tests := []struct {
		name    string
		args    args
		want    *types.Subnet
		wantErr bool
	}{
		{
			name: "single subnet pool",
			args: args{
				subnetPools: []config.SubnetPool{
					{Base: parseIPNet("10.89.0.0/16"), Size: 24},
				},
			},
			want: parseSubnet("10.89.0.0/24"),
		},
		{
			name: "single subnet pool with used nets",
			args: args{
				subnetPools: []config.SubnetPool{
					{Base: parseIPNet("10.89.0.0/16"), Size: 24},
				},
				usedNetworks: []*net.IPNet{
					parseCIDR("10.89.0.0/25"),
				},
			},
			want: parseSubnet("10.89.1.0/24"),
		},
		{
			name: "single subnet pool with no free nets",
			args: args{
				subnetPools: []config.SubnetPool{
					{Base: parseIPNet("10.89.0.0/16"), Size: 24},
				},
				usedNetworks: []*net.IPNet{
					parseCIDR("10.89.0.0/16"),
				},
			},
			wantErr: true,
		},
		{
			name: "two subnet pools",
			args: args{
				subnetPools: []config.SubnetPool{
					{Base: parseIPNet("10.89.0.0/16"), Size: 24},
					{Base: parseIPNet("10.90.0.0/16"), Size: 25},
				},
			},
			want: parseSubnet("10.89.0.0/24"),
		},
		{
			name: "two subnet pools with no free subnet in first pool",
			args: args{
				subnetPools: []config.SubnetPool{
					{Base: parseIPNet("10.89.0.0/16"), Size: 24},
					{Base: parseIPNet("10.90.0.0/16"), Size: 25},
				},
				usedNetworks: []*net.IPNet{
					parseCIDR("10.89.0.0/16"),
				},
			},
			want: parseSubnet("10.90.0.0/25"),
		},
		{
			name: "two subnet pools with no free subnet",
			args: args{
				subnetPools: []config.SubnetPool{
					{Base: parseIPNet("10.89.0.0/16"), Size: 24},
					{Base: parseIPNet("10.90.0.0/16"), Size: 25},
				},
				usedNetworks: []*net.IPNet{
					parseCIDR("10.89.0.0/8"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFreeIPv4NetworkSubnet(tt.args.usedNetworks, tt.args.subnetPools)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFreeIPv4NetworkSubnet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFreeIPv4NetworkSubnet() = %v, want %v", got.Subnet.String(), tt.want.Subnet.String())
			}
		})
	}
}
