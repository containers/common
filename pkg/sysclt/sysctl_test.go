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
