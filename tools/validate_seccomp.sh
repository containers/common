#!/bin/bash
#
# validate_seccomp.sh <gopath/to/pkg/seccomp>
#
# Validates that the seccomp.json file has been generated and matches the
# profile defined in the pkg/seccomp package.

set -Eeuo pipefail

PACKAGE_PATH="${1:-./pkg/seccomp}"
TARGET_FILE="$PACKAGE_PATH/seccomp.json"

# Stash a copy.
tmp_copy="$(mktemp --tmpdir podman-seccomp.json.XXXXXX)"
cp "$TARGET_FILE" "$tmp_copy"

# Generate it again and figure out if there was a difference.
go generate -tags seccomp "$PACKAGE_PATH" >/dev/null
diffs="$(diff -u "$tmp_copy" "$TARGET_FILE" ||:)"

if [ "$diffs" ]; then
	# Can we make a prettier diff?
	have_diffstat=1
	which diffstat || have_diffstat=
	if [ "$have_diffstat" ]; then
		diffs="$(echo "$diffs" | diffstat)"
	fi

	# Output an error message and fail the CI.
	cat >&2 <<-EOF
	The result of 'go generate -tags seccomp $PACKAGE_PATH' differs.

	$diffs

	Please re-run 'go generate -tags seccomp $PACKAGE_PATH' and then amend your
	commits to include the updated seccomp.json file.
	EOF
	exit 1
fi
