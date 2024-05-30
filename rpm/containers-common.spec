# Below definitions are used to deliver config files from a particular branch
# of c/image, c/storage  and c/shortnames vendored in all of Buildah, Podman and Skopeo.
# These vendored components must have the same version. If it is not the case,
# pick the oldest version on c/image, c/storage and c/shortnames vendored in
# Buildah/Podman/Skopeo.

# Packit will automatically update the image and storage versions on Fedora and
# CentOS Stream dist-git PRs.
%global image_branch main
%global storage_branch main
%global shortnames_branch main

%global project containers
%global repo common

%global raw_github_url https://raw.githubusercontent.com/%{project}

%if %{defined copr_username}
%define copr_build 1
%endif

Name: containers-common
%if %{defined copr_build}
Epoch: 102
%else
Epoch: 5
%endif
# DO NOT TOUCH the Version string!
# The TRUE source of this specfile is:
# https://github.com/containers/common/blob/main/rpm/containers-common.spec
# If that's what you're reading, Version must be 0, and will be updated by Packit for
# copr and koji builds.
# If you're reading this on dist-git, the version is automatically filled in by Packit.
Version: 0
Release: %autorelease
License: Apache-2.0
BuildArch: noarch
# for BuildRequires: go-md2man
ExclusiveArch: %{golang_arches} noarch
Summary: Common configuration and documentation for containers
BuildRequires: git-core
BuildRequires: go-md2man
Provides: skopeo-containers = %{epoch}:%{version}-%{release}
Requires: (container-selinux >= 2:2.162.1 if selinux-policy)
Suggests: fuse-overlayfs
URL: https://github.com/%{project}/%{repo}
Source0: %{url}/archive/v%{version_no_tilde}.tar.gz
Source1: %{raw_github_url}/image/%{image_branch}/docs/containers-auth.json.5.md
Source2: %{raw_github_url}/image/%{image_branch}/docs/containers-certs.d.5.md
Source3: %{raw_github_url}/image/%{image_branch}/docs/containers-policy.json.5.md
Source4: %{raw_github_url}/image/%{image_branch}/docs/containers-registries.conf.5.md
Source5: %{raw_github_url}/image/%{image_branch}/docs/containers-registries.conf.d.5.md
Source6: %{raw_github_url}/image/%{image_branch}/docs/containers-registries.d.5.md
Source7: %{raw_github_url}/image/%{image_branch}/docs/containers-signature.5.md
Source8: %{raw_github_url}/image/%{image_branch}/docs/containers-transports.5.md
Source9: %{raw_github_url}/storage/%{storage_branch}/docs/containers-storage.conf.5.md
Source10: %{raw_github_url}/shortnames/%{shortnames_branch}/shortnames.conf
Source11: %{raw_github_url}/image/%{image_branch}/default.yaml
Source12: %{raw_github_url}/image/%{image_branch}/default-policy.json
Source13: %{raw_github_url}/image/%{image_branch}/registries.conf
Source14: %{raw_github_url}/storage/%{storage_branch}/storage.conf
# Fetch RPM-GPG-KEY-redhat-release from the authoritative source instead of storing
# a copy in repo or dist-git. Depending on distribution-gpg-keys rpm is also
# not an option because that package doesn't exist on CentOS Stream.
Source15: https://access.redhat.com/security/data/fd431d51.txt

%description
This package contains common configuration files and documentation for container
tools ecosystem, such as Podman, Buildah and Skopeo.

It is required because the most of configuration files and docs come from projects
which are vendored into Podman, Buildah, Skopeo, etc. but they are not packaged
separately.

%package extra
Summary: Extra dependencies for Podman and Buildah
Requires: %{name} = %{epoch}:%{version}-%{release}
Requires: container-network-stack
Requires: oci-runtime
Conflicts: podman < 5:5.0.0~rc4-1
Recommends: composefs
Recommends: crun
Requires: (crun if fedora-release-identity-server)
Requires: netavark >= 1.10.3-1
Suggests: slirp4netns
Requires: passt
Requires: iptables
Requires: nftables
Recommends: qemu-user-static
Requires: (qemu-user-static-aarch64 if fedora-release-identity-server)
Requires: (qemu-user-static-arm if fedora-release-identity-server)
Requires: (qemu-user-static-x86 if fedora-release-identity-server)

%description extra
This subpackage will handle dependencies common to Podman and Buildah which are
not required by Skopeo.

%prep
%autosetup -Sgit -n %{repo}-%{version_no_tilde}

# Copy manpages to docs subdir in builddir to build before installing.
cp %{SOURCE1} docs/.
cp %{SOURCE2} docs/.
cp %{SOURCE3} docs/.
cp %{SOURCE4} docs/.
cp %{SOURCE5} docs/.
cp %{SOURCE6} docs/.
cp %{SOURCE7} docs/.
cp %{SOURCE8} docs/.
cp %{SOURCE9} docs/.

# Copy config files to builddir to patch them before installing.
# Currently, only registries.conf and storage.conf files are patched before
# installing.
cp %{SOURCE10} shortnames.conf
cp %{SOURCE13} registries.conf
cp %{SOURCE14} storage.conf

# Fine-grain distro- and release-specific tuning of config files,
# e.g., seccomp, composefs, registries on different RHEL/Fedora versions
bash rpm/update-config-files.sh

%build
mkdir -p man5
for i in docs/*.5.md; do
    go-md2man -in $i -out man5/$(basename $i .md)
done

%install
# install config and policy files for registries
install -dp %{buildroot}%{_sysconfdir}/containers/{certs.d,oci/hooks.d,systemd}
install -dp %{buildroot}%{_sharedstatedir}/containers/sigstore
install -dp %{buildroot}%{_datadir}/containers/systemd
install -dp %{buildroot}%{_prefix}/lib/containers/storage
install -dp -m 700 %{buildroot}%{_prefix}/lib/containers/storage/overlay-images
touch %{buildroot}%{_prefix}/lib/containers/storage/overlay-images/images.lock
install -dp -m 700 %{buildroot}%{_prefix}/lib/containers/storage/overlay-layers
touch %{buildroot}%{_prefix}/lib/containers/storage/overlay-layers/layers.lock

install -Dp -m0644 shortnames.conf %{buildroot}%{_sysconfdir}/containers/registries.conf.d/000-shortnames.conf
install -Dp -m0644 %{SOURCE11} %{buildroot}%{_sysconfdir}/containers/registries.d/default.yaml
install -Dp -m0644 %{SOURCE12} %{buildroot}%{_sysconfdir}/containers/policy.json
install -Dp -m0644 registries.conf %{buildroot}%{_sysconfdir}/containers/registries.conf
install -Dp -m0644 storage.conf %{buildroot}%{_datadir}/containers/storage.conf

# RPM-GPG-KEY-redhat-release already exists on rhel envs, install only on
# fedora and centos
%if %{defined fedora} || %{defined centos}
install -Dp -m0644 %{SOURCE15} %{buildroot}%{_sysconfdir}/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
%endif

install -Dp -m0644 contrib/redhat/registry.access.redhat.com.yaml -t %{buildroot}%{_sysconfdir}/containers/registries.d
install -Dp -m0644 contrib/redhat/registry.redhat.io.yaml -t %{buildroot}%{_sysconfdir}/containers/registries.d

# install manpages
install -dp %{buildroot}%{_mandir}/man5
for i in man5/*.5; do
   install -Dp -m0644 $i -t %{buildroot}%{_mandir}/man5
done
ln -s containerignore.5 %{buildroot}%{_mandir}/man5/.containerignore.5

# install config files for mounts, containers and seccomp
install -m0644 pkg/subscriptions/mounts.conf %{buildroot}%{_datadir}/containers/mounts.conf
install -m0644 pkg/seccomp/seccomp.json %{buildroot}%{_datadir}/containers/seccomp.json
install -m0644 pkg/config/containers.conf %{buildroot}%{_datadir}/containers/containers.conf

# install secrets patch directory
install -d -p -m 755 %{buildroot}/%{_datadir}/rhel/secrets
# rhbz#1110876 - update symlinks for subscription management
ln -s ../../../..%{_sysconfdir}/pki/entitlement %{buildroot}%{_datadir}/rhel/secrets/etc-pki-entitlement
ln -s ../../../..%{_sysconfdir}/rhsm %{buildroot}%{_datadir}/rhel/secrets/rhsm
ln -s ../../../..%{_sysconfdir}/yum.repos.d/redhat.repo %{buildroot}%{_datadir}/rhel/secrets/redhat.repo

%files
%dir %{_sysconfdir}/containers
%dir %{_sysconfdir}/containers/certs.d
%dir %{_sysconfdir}/containers/oci
%dir %{_sysconfdir}/containers/oci/hooks.d
%dir %{_sysconfdir}/containers/registries.conf.d
%dir %{_sysconfdir}/containers/registries.d
%dir %{_sysconfdir}/containers/systemd
%dir %{_prefix}/lib/containers/storage
%dir %{_prefix}/lib/containers/storage/overlay-images
%dir %{_prefix}/lib/containers/storage/overlay-layers
%{_prefix}/lib/containers/storage/overlay-images/images.lock
%{_prefix}/lib/containers/storage/overlay-layers/layers.lock

%config(noreplace) %{_sysconfdir}/containers/policy.json
%config(noreplace) %{_sysconfdir}/containers/registries.conf
%config(noreplace) %{_sysconfdir}/containers/registries.conf.d/000-shortnames.conf
%if 0%{?fedora} || 0%{?centos}
%{_sysconfdir}/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
%endif
%config(noreplace) %{_sysconfdir}/containers/registries.d/default.yaml
%config(noreplace) %{_sysconfdir}/containers/registries.d/registry.redhat.io.yaml
%config(noreplace) %{_sysconfdir}/containers/registries.d/registry.access.redhat.com.yaml
%ghost %{_sysconfdir}/containers/storage.conf
%ghost %{_sysconfdir}/containers/containers.conf
%dir %{_sharedstatedir}/containers/sigstore
%{_mandir}/man5/Containerfile.5.gz
%{_mandir}/man5/containerignore.5.gz
%{_mandir}/man5/.containerignore.5.gz
%{_mandir}/man5/containers*.5.gz
%dir %{_datadir}/containers
%dir %{_datadir}/containers/systemd
%{_datadir}/containers/storage.conf
%{_datadir}/containers/containers.conf
%{_datadir}/containers/mounts.conf
%{_datadir}/containers/seccomp.json
%dir %{_datadir}/rhel/secrets
%{_datadir}/rhel/secrets/*

%files extra

%changelog
%autochangelog
