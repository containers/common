package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/containers/common/libnetwork/types"
)

type Error struct {
	Msg string `json:"error"`
}

func errAndExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		errAndExit("netavark testplugin: argument required: [create|setup|teardown]")
	}

	switch os.Args[1] {
	case "create":
		if err := create(); err != nil {
			out := Error{Msg: err.Error()}
			_ = json.NewEncoder(os.Stdout).Encode(out)
			os.Exit(1)
		}
	case "setup", "teardown":
		// this is executed and tested in netavark so we not need it here
	default:
		errAndExit(fmt.Sprintf("unknown argument: %s", os.Args[1]))
	}
}

func create() error {
	network := types.Network{}
	d := json.NewDecoder(os.Stdin)
	err := d.Decode(&network)
	if err != nil {
		return fmt.Errorf("failed to decode network input: %w", err)
	}

	// for testing purpose error out when tests set error option
	msg, ok := network.Options["error"]
	if ok {
		return fmt.Errorf("%s", msg)
	}

	// for testing purpose change field when instructed to do so
	name, ok := network.Options["name"]
	if ok {
		network.Name = name
	}
	id, ok := network.Options["id"]
	if ok {
		network.ID = id
	}
	driver, ok := network.Options["driver"]
	if ok {
		network.Driver = driver
	}

	e := json.NewEncoder(os.Stdout)
	return e.Encode(network)
}
