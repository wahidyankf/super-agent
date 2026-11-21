#!/usr/bin/env bash
# Commitlint receives each commit in the immutable base..head range on its own. A concatenated message stream cannot
# show which commit broke the format.
set -euo pipefail

[[ $# -eq 2 ]] || {
	printf '%s\n' 'check-commit-range.sh requires a base and head revision' >&2
	exit 2
}

exec npm exec -- commitlint --from "$1" --to "$2" --verbose
