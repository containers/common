module github.com/containers/common

go 1.15

require (
	github.com/BurntSushi/toml v1.0.0
	github.com/containerd/containerd v1.6.1
	github.com/containernetworking/cni v1.0.1
	github.com/containernetworking/plugins v1.1.1
	github.com/containers/image/v5 v5.19.2-0.20220224100137-1045fb70b094
	github.com/containers/ocicrypt v1.1.3
	github.com/containers/storage v1.38.2
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/disiqueira/gotree/v3 v3.0.2
	github.com/docker/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.14+incompatible
	github.com/docker/go-units v0.4.0
	github.com/ghodss/yaml v1.0.0
	github.com/godbus/dbus/v5 v5.1.0
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jinzhu/copier v0.3.5
	github.com/json-iterator/go v1.1.12
	github.com/mitchellh/mapstructure v1.4.3
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.3-0.20211202193544-a5463b7f9c84
	github.com/opencontainers/runc v1.1.0
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/opencontainers/runtime-tools v0.9.0
	github.com/opencontainers/selinux v1.10.0
	github.com/pkg/errors v0.9.1
	github.com/seccomp/libseccomp-golang v0.9.2-0.20210429002308-3879420cc921
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.1
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635
	github.com/vishvananda/netlink v1.1.1-0.20210330154013-f5de75959ad5
	go.etcd.io/bbolt v1.3.6
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220128215802-99c3d69c2c27
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
)

retract (
	v1.0.1 // This version is used only to publish retraction of v1.0.1.
	v1.0.0 // We reverted to v0.… version numbers; the v1.0.0 tag was actually deleted.
)
