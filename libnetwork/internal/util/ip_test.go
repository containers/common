package util

import (
	"fmt"
	"net"
	"reflect"
	"testing"
)

func parseCIDR(n string) *net.IPNet {
	_, parsedNet, _ := net.ParseCIDR(n)
	return parsedNet
}

func TestNextSubnet(t *testing.T) {
	type args struct {
		subnet *net.IPNet
	}
	tests := []struct {
		name    string
		args    args
		want    *net.IPNet
		wantErr bool
	}{
		{"class a", args{subnet: parseCIDR("10.0.0.0/8")}, parseCIDR("11.0.0.0/8"), false},
		{"class b", args{subnet: parseCIDR("192.168.0.0/16")}, parseCIDR("192.169.0.0/16"), false},
		{"class c", args{subnet: parseCIDR("192.168.1.0/24")}, parseCIDR("192.168.2.0/24"), false},
		{"custom cidr /23", args{subnet: parseCIDR("192.168.0.0/23")}, parseCIDR("192.168.2.0/23"), false},
		{"custom cidr /23 2", args{subnet: parseCIDR("192.168.2.0/23")}, parseCIDR("192.168.4.0/23"), false},
		{"custom cidr /23 with overflow", args{subnet: parseCIDR("192.168.254.0/23")}, parseCIDR("192.169.0.0/23"), false},
		{"/9", args{subnet: parseCIDR("10.0.0.0/9")}, parseCIDR("10.128.0.0/9"), false},
		{"/11", args{subnet: parseCIDR("10.0.0.0/11")}, parseCIDR("10.32.0.0/11"), false},
		{"/11", args{subnet: parseCIDR("0.0.0.0/11")}, parseCIDR("0.32.0.0/11"), false},
		{"only one subnet", args{subnet: parseCIDR("0.0.0.0/0")}, nil, true},
		{"no more subnets", args{subnet: parseCIDR("255.255.255.0/24")}, nil, true},
		{"no more subnets 2", args{subnet: parseCIDR("255.240.0.0/12")}, nil, true},
		{"ipv6", args{subnet: parseCIDR("fdfd:d319:b145:fd4b::/64")}, parseCIDR("fdfd:d319:b145:fd4c::/64"), false},
		{"ipv6 with overflow", args{subnet: parseCIDR("fdfd:d319:b145:ffff::/64")}, parseCIDR("fdfd:d319:b146:0::/64"), false},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			got, err := NextSubnet(test.args.subnet)
			if (err != nil) != test.wantErr {
				t.Errorf("NextSubnet() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("NextSubnet() got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGetRandomIPv6Subnet(t *testing.T) {
	for i := 0; i < 1000; i++ {
		t.Run(fmt.Sprintf("GetRandomIPv6Subnet %d", i), func(t *testing.T) {
			sub, err := getRandomIPv6Subnet()
			if err != nil {
				t.Errorf("GetRandomIPv6Subnet() error should be nil: %v", err)
				return
			}
			if sub.IP.To4() != nil {
				t.Errorf("ip %s is not an ipv6 address", sub.IP)
			}
			if sub.IP[0] != 0xfd {
				t.Errorf("ipv6 %s does not start with fd", sub.IP)
			}
			ones, bytes := sub.Mask.Size()
			if ones != 64 || bytes != 128 {
				t.Errorf("wrong network mask %v, it should be /64", sub.Mask)
			}
		})
	}
}
