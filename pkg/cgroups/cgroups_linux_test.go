//go:build linux

package cgroups

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/containers/storage/pkg/unshare"
	"github.com/opencontainers/cgroups"
)

func TestCreated(t *testing.T) {
	// tests only works in rootless mode.
	if unshare.IsRootless() {
		return
	}

	var resources cgroups.Resources
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

	wtDevices := []*cgroups.WeightDevice{}
	devices := []*cgroups.ThrottleDevice{}
	dev1 := cgroups.NewThrottleDevice(1, 3, 2097152)
	dev2 := cgroups.NewThrottleDevice(3, 10, 2097152)
	dev3 := cgroups.NewWeightDevice(5, 9, 500, 0)
	devices = append(devices, dev1, dev2)
	wtDevices = append(wtDevices, dev3)

	var resources cgroups.Resources
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
	u, _, b, _, _, _ := resourcesToProps(&resources, true)

	val, ok := u["CPUQuotaPerSecUSec"]
	if !ok {
		t.Fatal("CPU Quota not parsed.")
	}
	if val != 1000000 {
		t.Fatal("CPU Quota incorrect value expected 1000000 got " + strconv.FormatUint(val, 10))
	}

	bits := new(big.Int)
	cpuset_val := bits.SetBit(bits, 0, 1).Bytes()

	cpus, ok := b["AllowedCPUs"]
	if !ok {
		t.Fatal("Cpuset Cpus not parsed.")
	}
	if !bytes.Equal(cpus, cpuset_val) {
		t.Fatal("Cpuset Cpus incorrect value expected " + string(cpuset_val) + " got " + string(cpus))
	}

	mems, ok := b["AllowedMemoryNodes"]
	if !ok {
		t.Fatal("Cpuset Mems not parsed.")
	}
	if !bytes.Equal(mems, cpuset_val) {
		t.Fatal("Cpuset Mems incorrect value expected " + string(cpuset_val) + " got " + string(mems))
	}

	err = os.Mkdir("/dev/foodevdir", os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("/dev/foodevdir")

	c := exec.CommandContext(context.Background(), "mknod", "/dev/foodevdir/null", "b", "1", "3")
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		t.Fatal(err)
	}

	c = exec.CommandContext(context.Background(), "mknod", "/dev/foodevdir/bar", "b", "3", "10")
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		t.Fatal(err)
	}

	c = exec.CommandContext(context.Background(), "mknod", "/dev/foodevdir/bat", "b", "5", "9")
	c.Env = os.Environ()
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	err = c.Run()
	if err != nil {
		t.Fatal(err)
	}

	resources.BlkioThrottleReadBpsDevice = []*cgroups.ThrottleDevice{devices[0]}
	resources.BlkioThrottleWriteBpsDevice = []*cgroups.ThrottleDevice{devices[1]}
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
