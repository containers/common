#!/usr/bin/env bash

set -eox pipefail

RHEL_RELEASE=$(rpm --eval %{?rhel})
ARCH=$(uname -m)

if [ $RHEL_RELEASE -eq 8 ]; then
    dnf -y module disable container-tools
fi
if [ -f /etc/centos-release ]; then
    dnf -y install epel-release
elif [ $RHEL_RELEASE -ge 8 ]; then
    dnf -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-$RHEL_RELEASE.noarch.rpm
    dnf config-manager --set-enabled epel
    cat /etc/yum.repos.d/epel.repo
fi
dnf -y copr enable rhcontainerbot/podman-next
dnf config-manager --save --setopt="*:rhcontainerbot:podman-next.priority=5"
