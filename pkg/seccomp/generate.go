//go:build ignore
// +build ignore

// Copyright 2013-2021 Docker, Inc.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/containers/common/pkg/seccomp"
)

// saves the default seccomp profile as a json file so people can use it as a
// base for their own custom profiles
func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	f := filepath.Join(wd, "seccomp.json")

	// write the default profile to the file
	b, err := json.MarshalIndent(seccomp.DefaultProfile(), "", "\t")
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(f, b, 0o644); err != nil {
		panic(err)
	}
}
