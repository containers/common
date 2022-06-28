//go:build linux
// +build linux

package cgroups

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/containers/storage/pkg/unshare"
	"github.com/opencontainers/runc/libcontainer/configs"
)

func TestCreated(t *testing.T) {
	// tests only works in rootless mode.
	if unshare.IsRootless() {
		return
	}

	var resources configs.Resources
	cgr, err := New("machine.slice", &resources)
	if err != nil {
		t.Fatal(err)
	}
	if err := cgr.Delete(); err != nil {
		t.Fatal(err)
	}

	cgr, err = NewSystemd("machine.slice", &resources)
	if err != nil {
		t.Fatal(err)
	}
	if err := cgr.Delete(); err != nil {
		t.Fatal(err)
	}
}

func TestResources(t *testing.T) {
	// tests only works in rootful mode.
	if unshare.IsRootless() {
		return
	}

	var resources configs.Resources
	resources.CpuPeriod = 100000
	resources.CpuQuota = 100000
	resources.Memory = 900
	resources.MemorySwap = 1000
	resources.BlkioWeight = 300

	cgr, err := New("machine.slice", &resources)
	if err != nil {
		t.Fatal(err)
	}

	// TestMode is used in the runc packages for unit tests, works without this as well here.
	TestMode = true
	err = cgr.Update(&resources)
	if err != nil {
		t.Fatal(err)
	}
	if cgr.config.CpuPeriod != 100000 || cgr.config.CpuQuota != 100000 {
		t.Fatal("Got the wrong value, set cpu.cfs_period_us failed.")
	}

	if err := cgr.Delete(); err != nil {
		t.Fatal(err)
	}

	cgr2, err := NewSystemd("machine2.slice", &resources)
	if err != nil {
		t.Fatal(err)
	}

	// test CPU Quota adjustment.
	u, _, _, _ := resourcesToProps(&resources)

	val, ok := u["CPUQuotaPerSecUSec"]
	if !ok {
		t.Fatal("CPU Quota not parsed.")
	}
	if val != 1000000 {
		t.Fatal("CPU Quota incorrect value expected 1000000 got " + strconv.FormatUint(val, 10))
	}

	// machine.slice = parent, libpod_pod_ID = path
	err = cgr2.CreateSystemdUnit(fmt.Sprintf("%s/%s-%s%s", "machine2.slice", "machine2", "libpod_pod_1234", ".slice"))
	if err != nil {
		t.Fatal(err)
	}

	if err := cgr2.Delete(); err != nil {
		t.Fatal(err)
	}
}
