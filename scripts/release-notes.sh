#!/usr/bin/env sh
# Print the CHANGELOG.md section for a version, to use as GitHub release notes.
#
# Usage: scripts/release-notes.sh 0.1.0
#
# Fails when the version has no section, so a tag pushed before its changelog
# entry exists cannot publish a release with empty notes.
set -eu

version="${1:?usage: release-notes.sh <version>}"
changelog="${2:-CHANGELOG.md}"

notes="$(awk -v version="$version" '
	/^## \[/ {
		if (found) exit
		if (index($0, "## [" version "]") == 1) { found = 1; next }
	}
	found { print }
' "$changelog")"

# Drop leading and trailing blank lines.
notes="$(printf '%s\n' "$notes" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')"

if [ -z "$notes" ]; then
	echo "error: no section for version $version in $changelog" >&2
	exit 1
fi

printf '%s\n' "$notes"
