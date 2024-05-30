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
ensure pkg/config/containers.conf   compression_format  \"zstd:chunked\"

# Enable seccomp support keyctl and socketcall
grep -q \"keyctl\", pkg/seccomp/seccomp.json || sed -i '/\"kill\",/i \
    "keyctl",' pkg/seccomp/seccomp.json
grep -q \"socket\", pkg/seccomp/seccomp.json || sed -i '/\"socketcall\",/i \
    "socket",' pkg/seccomp/seccomp.json

FEDORA=$(rpm --eval '%{?fedora}')
RHEL=$(rpm --eval '%{?rhel}')
COPR=$(rpm --eval '%{?copr_username}')

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

# Set these on Fedora Rawhide (41+), RHEL 10+, and on all COPR builds
# regardless of distro
if [[ -n "$COPR" ]] || [[ "$FEDORA" -gt 40 ]] || [[ "$RHEL" -ge 10 ]]; then
    ensure storage.conf pull_options    \{enable_partial_images\ =\ \"true\",\ use_hard_links\ =\ \"false\",\ ostree_repos=\"\",\ convert_images\ =\ \"false\"\}
    # Leave composefs disabled
    ensure storage.conf use_composefs   \"false\"
fi
