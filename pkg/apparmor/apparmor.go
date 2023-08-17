package apparmor

import (
	"errors"
)

const (
	// ProfilePrefix is used for version-independent presence checks.
	ProfilePrefix = "containers-default-"

	// Default AppArmor profile used by containers; by default this is set to unconfined.
	// To override this, distros should supply their own profile and specify it in a default
	// containers.conf.
	// See the following issues for more information:
	// - https://github.com/containers/common/issues/958
	// - https://github.com/containers/podman/issues/15874
	Profile = "unconfined"
)

var (
	// ErrApparmorUnsupported indicates that AppArmor support is not supported.
	ErrApparmorUnsupported = errors.New("AppArmor is not supported")
	// ErrApparmorRootless indicates that AppArmor support is not supported in rootless mode.
	ErrApparmorRootless = errors.New("AppArmor is not supported in rootless mode")
)
