#!/usr/bin/env bash
# This script delivers current documentation/configs and assures it has the intended
# settings for a particular branch/release.

set -exo pipefail

ensure() {
  if [[ ! -f $1 ]]; then
      echo "File not found:" $1
      exit 1
  fi
  if grep ^$2[[:blank:]].*= $1 > /dev/null
  then
    sed -i "s;^$2[[:blank:]]=.*;$2 = $3;" $1
  else
    if grep ^\#.*$2[[:blank:]].*= $1 > /dev/null
    then
      sed -i "/^#.*$2[[:blank:]].*=/a \
$2 = $3" $1
    else
      echo "$2 = $3" >> $1
    fi
  fi
}

# Common options enabled across all fedora, centos, rhel
# TBD: Can these be enabled by default upstream?
ensure registries.conf              short-name-mode     \"enforcing\"

ensure storage.conf                 driver              \"overlay\"
ensure storage.conf                 mountopt            \"nodev,metacopy=on\"

ensure pkg/config/containers.conf   runtime             \"crun\"
ensure pkg/config/containers.conf   log_driver          \"journald\"

FEDORA=$(rpm --eval '%{?fedora}')
RHEL=$(rpm --eval '%{?rhel}')

# Set search registries
if [[ -n "$FEDORA" ]]; then
    ensure registries.conf unqualified-search-registries [\"registry.fedoraproject.org\",\ \"registry.access.redhat.com\",\ \"docker.io\"]
else
    ensure registries.conf unqualified-search-registries [\"registry.access.redhat.com\",\ \"registry.redhat.io\",\ \"docker.io\"]
fi

# Set these on all Fedora and RHEL 10+
if [[ -n "$FEDORA" ]] || [[ "$RHEL" -ge 10 ]]; then
    sed -i -e '/^additionalimagestores\ =\ \[/a "\/usr\/lib\/containers\/storage",' storage.conf
fi
