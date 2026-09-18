package sysctl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	strSlice := []string{"net.core.test1=4", "kernel.msgmax=2"}
	result, err := Validate(strSlice)
	require.Nil(t, err)
	assert.Equal(t, result["net.core.test1"], "4")
}

func TestValidateBadSysctl(t *testing.T) {
	strSlice := []string{"BLAU=BLUE", "GELB^YELLOW"}
	_, err := Validate(strSlice)
	assert.Error(t, err)
}

func TestConvertSysctlVariableToDotsSeparator(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{in: "kernel.shm_rmid_forced", out: "kernel.shm_rmid_forced"},
		{in: "kernel/shm_rmid_forced", out: "kernel.shm_rmid_forced"},
		{in: "net.ipv4.conf.eno2/100.rp_filter", out: "net.ipv4.conf.eno2/100.rp_filter"},
		{in: "net/ipv4/conf/eno2.100/rp_filter", out: "net.ipv4.conf.eno2/100.rp_filter"},
		{in: "net/ipv6/conf/bond1.340/autoconf", out: "net.ipv6.conf.bond1/340.autoconf"},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.out, convertSysctlVariableToDotsSeparator(tc.in))
	}
}

func TestValidateSlashSeparatedSysctl(t *testing.T) {
	strSlice := []string{
		"net/ipv6/conf/bond1.340/autoconf=0",
		"net.ipv6.conf.bond1/340.autoconf=0",
	}
	result, err := Validate(strSlice)
	require.Nil(t, err)
	assert.Equal(t, "0", result["net/ipv6/conf/bond1.340/autoconf"])
	assert.Equal(t, "0", result["net.ipv6.conf.bond1/340.autoconf"])
}
