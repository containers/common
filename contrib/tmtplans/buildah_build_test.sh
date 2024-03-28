#!/usr/bin/env bash

set -eox pipefail

rpm -q golang

if [ -f /etc/fedora-release ]; then
    export TMPDIR=/var/tmp
fi

git clone https://github.com/containers/podman

cd podman
dnf -y builddep rpm/podman.spec

go mod edit -replace github.com/containers/common=../
make vendor
cat go.mod

git add vendor/
git config --global user.email "you@example.com"
git config --global user.name "Your Name"

make rpm
