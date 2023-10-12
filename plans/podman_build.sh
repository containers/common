#!/usr/bin/env bash

set -eox pipefail

cat /etc/redhat-release
git clone https://github.com/containers/podman
cd podman
dnf -y builddep rpm/podman.spec
go mod edit -require github.com/containers/common@HEAD
make vendor
make rpm
