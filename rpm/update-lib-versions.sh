#!/usr/bin/env bash

set -exo pipefail

SPECFILE=rpm/containers-common.spec

# Fetch versions from go.mod
IMAGE_VERSION=$(awk '/github.com\/containers\/image/ {print $2}' go.mod)
STORAGE_VERSION=$(awk '/github.com\/containers\/storage/ {print $2}' go.mod)

# Update versions in rpm spec
sed -i "s/^%global image_branch main/%global image_branch $IMAGE_VERSION/" $SPECFILE
sed -i "s/^%global storage_branch main/%global storage_branch $STORAGE_VERSION/" $SPECFILE
