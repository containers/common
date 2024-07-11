#!/usr/bin/env bash

set -exo pipefail

mkdir -p man5
for i in docs/*.5.md; do
    go-md2man -in $i -out man5/$(basename $i .md)
done
