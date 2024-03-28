#!/usr/bin/env bash

set -exo pipefail

rpm -q golang

#if [ -f /etc/fedora-release ]; then
#    export TMPDIR=/var/tmp
#fi

# Navigate to parent dir of default working dir
cd ..

# Clone podman
git clone https://github.com/containers/podman

cd podman
dnf -y builddep rpm/podman.spec

# Vendor c/common from PR
# TMT_TREE points to the default working dir
go mod edit -replace github.com/containers/common=$TMT_TREE
make vendor
cat go.mod

make
