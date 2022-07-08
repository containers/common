//go:build linux
// +build linux

package cgroups

import (
	"fmt"
	"os"
	"os/exec"
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

	wtDevices := []*configs.WeightDevice{}
	devices := []*configs.ThrottleDevice{}
	dev1 := &configs.ThrottleDevice{
		BlockIODevice: configs.BlockIODevice{
			Major: 1,
			Minor: 3,
		},
		Rate: 2097152,
	}
	dev2 := &configs.ThrottleDevice{
		BlockIODevice: configs.BlockIODevice{
			Major: 3,
			Minor: 10,
		},
		Rate: 2097152,
	}
	dev3 := &configs.WeightDevice{
		BlockIODevice: configs.BlockIODevice{
			Major: 5,
			Minor: 9,
		},
		Weight: 500,
	}
	devices = append(devices, dev1, dev2)
	wtDevices = append(wtDevices, dev3)

	var resources configs.Resources
	resources.CpuPeriod = 100000
	resources.CpuQuota = 100000
	resources.CpuShares = 100
	resources.CpusetCpus = "0"
	resources.CpusetMems = "0"
	resources.Memory = 900
	resources.MemorySwap = 1000
	resources.BlkioWeight = 300

	cgr, err := New("machine.slice", &resources)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := cgr.Delete(); err != nil {
			t.Fatal(err)
		}
	}()

	// TestMode is used in the runc packages for unit tests, works without this as well here.
	TestMode = true
	err = cgr.Update(&resources)
	if err != nil {
		t.Fatal(err)
	}
	if cgr.config.CpuPeriod != 100000 || cgr.config.CpuQuota != 100000 {
		t.Fatal("Got the wrong value, set cpu.cfs_period_us failed.")
	}

	cgr2, err := NewSystemd("machine2.slice", &resources)
	if err != nil {
		t.Fatal(err)
	}

	// test CPU Quota adjustment.
	u, _, _, _, _ := resourcesToProps(&resources, true)

	val, ok := u["CPUQuotaPerSecUSec"]
	if !ok {
		t.Fatal("CPU Quota not parsed.")
	}
	if val != 1000000 {
		t.Fatal("CPU Quota incorrect value expected 1000000 got " + strconv.FormatUint(val, 10))
	}

	err = os.Mkdir("/dev/foodevdir", os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/dev/foodevdir")

	c := exec.Command("mknod", "/dev/foodevdir/null", "b", "1", "3")
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		t.Fatal(err)
	}

	c = exec.Command("mknod", "/dev/foodevdir/bar", "b", "3", "10")
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		t.Fatal(err)
	}

	c = exec.Command("mknod", "/dev/foodevdir/bat", "b", "5", "9")
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		t.Fatal(err)
	}

	resources.BlkioThrottleReadBpsDevice = []*configs.ThrottleDevice{devices[0]}
	resources.BlkioThrottleWriteBpsDevice = []*configs.ThrottleDevice{devices[1]}
	resources.BlkioWeightDevice = wtDevices

	// machine.slice = parent, libpod_pod_ID = path
	err = cgr2.CreateSystemdUnit(fmt.Sprintf("%s/%s-%s%s", "machine2.slice", "machine2", "libpod_pod_12345", ".slice"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := cgr2.Delete(); err != nil {
			t.Fatal(err)
		}
	}()
}
