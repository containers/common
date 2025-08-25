# containers/common

> [!WARNING]
> This package was moved; please update your references to use `go.podman.io/common` instead.
> New development of this project happens on https://github.com/containers/container-libs.
> For more information, check https://blog.podman.io/2025/08/upcoming-migration-of-three-containers-repositories-to-monorepo/.

Location for shared common files and common go code to manage those files in
github.com/containers repos.

The common files to one or more projects in the containers group will be kept in
this repository.

It will be up to the individual projects to include the files from this
repository.

## seccomp

The `seccomp` package in [`pkg/seccomp`](pkg/seccomp) is a set of Go libraries
used by container runtimes to generate and load seccomp mappings into the
kernel.

seccomp (short for secure computing mode) is a BPF based syscall filter language
and present a more conventional function-call based filtering interface that
should be familiar to, and easily adopted by, application developers.

### Building the seccomp.json file

The make target `make seccomp.json` generates the seccomp.json file, which
contains the allowed list of syscalls that can be used by container runtime
engines like [CRI-O][cri-o], [Buildah][buildah], [Podman][podman] and
[Docker][docker], and container runtimes like OCI [Runc][runc] to control the
syscalls available to containers.

[cri-o]: https://github.com/cri-o/cri-o
[buildah]: https://github.com/containers/buildah
[podman]: https://github.com/containers/podman
[docker]: https://github.com/moby/moby
[runc]: https://github.com/opencontainers/runc

## Supported build tags

- [`pkg/apparmor`](pkg/apparmor): `apparmor`, `linux`
- [`pkg/seccomp`](pkg/seccomp): `seccomp`
- [`pkg/config`](pkg/config): `darwin`, `remote`, `linux`, `systemd`
- [`pkg/sysinfo`](pkg/sysctl): `linux`, `solaris`, `windows`, `cgo`
- [`pkg/cgroupv2`](pkg/cgroupv2): `linux`

## [Contributing](CONTRIBUTING.md)

When developing this library, please use `make` (or `make … BUILDTAGS=…`) to
take advantage of the tests and validation.

## Contact

https://podman.io/community
