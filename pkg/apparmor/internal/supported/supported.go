package supported

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/containers/storage/pkg/unshare"
	runcaa "github.com/opencontainers/runc/libcontainer/apparmor"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

type ApparmorVerifier struct {
	impl verifierImpl
}

func NewAppArmorVerifier() *ApparmorVerifier {
	return &ApparmorVerifier{impl: &defaultVerifier{}}
}

// IsSupported returns nil if AppAmor is supported by the host system,
// otherwise an error
func (a *ApparmorVerifier) IsSupported() error {
	if a.impl.UnshareIsRootless() {
		return errors.New("AppAmor is not supported on rootless containers")
	}
	if !a.impl.RuncIsEnabled() {
		return errors.New("AppArmor not supported by the host system")
	}

	const (
		binary = "apparmor_parser"
		sbin   = "/sbin"
	)

	// `/sbin` is not always in `$PATH`, so we check it explicitly
	sbinBinaryPath := filepath.Join(sbin, binary)
	if _, err := a.impl.OsStat(sbinBinaryPath); err == nil {
		logrus.Debugf("Found %s binary in %s", binary, sbinBinaryPath)
		return nil
	}

	// Fallback to checking $PATH
	if path, err := a.impl.ExecLookPath(binary); err == nil {
		logrus.Debugf("Found %s binary in %s", binary, path)
		return nil
	}

	return errors.Errorf(
		"%s binary neither found in %s nor $PATH", binary, sbin,
	)
}

//counterfeiter:generate . verifierImpl
type verifierImpl interface {
	UnshareIsRootless() bool
	RuncIsEnabled() bool
	OsStat(name string) (os.FileInfo, error)
	ExecLookPath(file string) (string, error)
}

type defaultVerifier struct{}

func (d *defaultVerifier) UnshareIsRootless() bool {
	return unshare.IsRootless()
}

func (d *defaultVerifier) RuncIsEnabled() bool {
	return runcaa.IsEnabled()
}

func (d *defaultVerifier) OsStat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (d *defaultVerifier) ExecLookPath(file string) (string, error) {
	return exec.LookPath(file)
}
