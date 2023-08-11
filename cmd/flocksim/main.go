package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/containers/storage/pkg/lockfile"
)

// flocksim is a testing tool used by the config lock tests
func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <path to lock>\n", os.Args[0])
		os.Exit(1)
	}

	lock, err := lockfile.GetLockFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	lock.Lock()
	fmt.Println("acquired lock, hit enter to release")

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
	lock.Unlock()
}
