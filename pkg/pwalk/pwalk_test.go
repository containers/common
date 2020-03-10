package pwalk

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestWalk(t *testing.T) {
	var count uint32
	concurrency := runtime.NumCPU() * 2

	dir, total, err := prepareTestSet(3, 2, 1)
	if err != nil {
		t.Fatalf("dataset creation failed: %v", err)
	}
	defer os.RemoveAll(dir)

	err = WalkN(dir,
		func(_ string, _ os.FileInfo, _ error) error {
			atomic.AddUint32(&count, 1)
			return nil
		},
		concurrency)

	if err != nil {
		t.Errorf("Walk failed: %v", err)
	}
	if count != uint32(total) {
		t.Errorf("File count mismatch: found %d, expected %d", count, total)
	}

	t.Logf("concurrency: %d, files found: %d\n", concurrency, count)
}

func TestWalkManyErrors(t *testing.T) {
	var count uint32

	dir, total, err := prepareTestSet(3, 3, 2)
	if err != nil {
		t.Fatalf("dataset creation failed: %v", err)
	}
	defer os.RemoveAll(dir)

	max := uint32(total / 2)
	e42 := errors.New("42")
	err = Walk(dir,
		func(p string, i os.FileInfo, _ error) error {
			if atomic.AddUint32(&count, 1) > max {
				return e42
			}
			return nil
		})
	t.Logf("found %d of %d files", count, total)

	if err == nil {
		t.Errorf("Walk succeeded, but error is expected")
		if count != uint32(total) {
			t.Errorf("File count mismatch: found %d, expected %d", count, total)
		}
	}
}

func makeManyDirs(prefix string, levels, dirs, files int) (count int, err error) {
	for d := 0; d < dirs; d++ {
		var dir string
		dir, err = ioutil.TempDir(prefix, "d-")
		if err != nil {
			return
		}
		count++
		for f := 0; f < files; f++ {
			fi, err := ioutil.TempFile(dir, "f-")
			if err != nil {
				return count, err
			}
			fi.Close()
			count++
		}
		if levels == 0 {
			continue
		}
		var c int
		if c, err = makeManyDirs(dir, levels-1, dirs, files); err != nil {
			return
		}
		count += c
	}

	return
}

// prepareTestSet() creates a directory tree of shallow files,
// to be used for testing or benchmarking.
//
// Total dirs: dirs^levels + dirs^(levels-1) + ... + dirs^1
// Total files: total_dirs * files
func prepareTestSet(levels, dirs, files int) (dir string, total int, err error) {
	dir, err = ioutil.TempDir(".", "pwalk-test-")
	if err != nil {
		return
	}
	total, err = makeManyDirs(dir, levels, dirs, files)
	if err != nil && total > 0 {
		_ = os.RemoveAll(dir)
		dir = ""
		total = 0
		return
	}
	total++ // this dir

	return
}

type walkerFunc func(root string, walkFn WalkFunc) error

func genWalkN(n int) walkerFunc {
	return func(root string, walkFn WalkFunc) error {
		return WalkN(root, walkFn, n)
	}
}

func BenchmarkWalk(b *testing.B) {
	const (
		levels = 5 // how deep
		dirs   = 3 // dirs on each levels
		files  = 8 // files on each levels
	)

	benchmarks := []struct {
		name string
		walk filepath.WalkFunc
	}{
		{"Empty", cbEmpty},
		{"ReadFile", cbReadFile},
		{"ChownChmod", cbChownChmod},
		{"RandomSleep", cbRandomSleep},
	}

	walkers := []struct {
		name   string
		walker walkerFunc
	}{
		{"filepath.Walk", filepath.Walk},
		{"pwalk.Walk", Walk},
		// test WalkN with various values of N
		{"pwalk.Walk1", genWalkN(1)},
		{"pwalk.Walk2", genWalkN(2)},
		{"pwalk.Walk4", genWalkN(4)},
		{"pwalk.Walk8", genWalkN(8)},
		{"pwalk.Walk16", genWalkN(16)},
		{"pwalk.Walk32", genWalkN(32)},
		{"pwalk.Walk64", genWalkN(64)},
		{"pwalk.Walk128", genWalkN(128)},
		{"pwalk.Walk256", genWalkN(256)},
	}

	dir, total, err := prepareTestSet(levels, dirs, files)
	if err != nil {
		b.Fatalf("dataset creation failed: %v", err)
	}
	defer os.RemoveAll(dir)
	b.Logf("dataset: %d levels x %d dirs x %d files, total entries: %d", levels, dirs, files, total)

	for _, bm := range benchmarks {
		for _, w := range walkers {
			// preheat
			err := w.walker(dir, bm.walk)
			if err != nil {
				b.Errorf("walk failed: %v", err)
			}
			// benchmark
			b.Run(bm.name+"/"+w.name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					err := w.walker(dir, bm.walk)
					if err != nil {
						b.Errorf("walk failed: %v", err)
					}
				}
			})
		}
	}
}
func cbEmpty(_ string, _ os.FileInfo, _ error) error {
	return nil
}

func cbChownChmod(path string, info os.FileInfo, _ error) error {
	_ = os.Chown(path, 0, 0)
	mode := os.FileMode(0644)
	if info.Mode().IsDir() {
		mode = os.FileMode(0755)
	}
	_ = os.Chmod(path, mode)

	return nil
}

func cbReadFile(path string, info os.FileInfo, _ error) error {
	var err error
	if info.Mode().IsRegular() {
		_, err = ioutil.ReadFile(path)
	}
	return err
}

func cbRandomSleep(_ string, _ os.FileInfo, _ error) error {
	time.Sleep(time.Duration(rand.Intn(500)) * time.Microsecond)
	return nil
}
