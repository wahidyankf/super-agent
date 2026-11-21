#!/bin/sh
# Health check for the Git hooks (C3). It exits 0 when core.hooksPath names the Husky runtime directory and the three
# hook files the registry wires are present and executable. Otherwise it prints one line per failed condition and exits
# 1. `npm install` installs the hooks through the prepare script. CI runs `npm ci --ignore-scripts`, which installs none,
# so this check is a bootstrap check for a contributor's checkout and is not a pull-request gate.
#
# Usage: scripts/check-hooks-installed.sh [repository-root]
set -eu

root=${1:-$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)}
expected_hooks_path='.husky/_'
status=0

# --local reads the repository's own configuration, so a global core.hooksPath cannot pass the check.
configured=$(git -C "$root" config --local --get core.hooksPath 2>/dev/null || true)
if [ "$configured" != "$expected_hooks_path" ]; then
	printf 'FAIL: core.hooksPath is "%s", expected "%s". Run npm install to install the hooks.\n' \
		"$configured" "$expected_hooks_path" >&2
	status=1
fi

for hook in pre-commit commit-msg pre-push; do
	file="$root/.husky/$hook"
	if [ ! -f "$file" ]; then
		printf 'FAIL: hook file .husky/%s is missing.\n' "$hook" >&2
		status=1
	elif [ ! -x "$file" ]; then
		printf 'FAIL: hook file .husky/%s is not executable.\n' "$hook" >&2
		status=1
	fi
done

if [ "$status" -eq 0 ]; then
	printf '%s\n' 'Git hooks: installed'
fi
exit "$status"
