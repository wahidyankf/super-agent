#!/bin/sh
# Self-test for scripts/check-hooks-installed.sh. Each case builds a fixture repository in a temporary directory, sets
# the hook configuration and files it needs, and compares the check's exit status and output with the expected ones.
# No test reads the real checkout's hooks, so the test passes whether or not the hooks are installed here.
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
check="$here/check-hooks-installed.sh"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cases=0
failed=0

# A Git hook hands its repository to every child process through these variables. Drop them, so each fixture is its own
# repository and not the one whose hook started this test.
for variable in $(git rev-parse --local-env-vars 2>/dev/null || true); do
	unset "$variable"
done

# make_fixture makes the repository $work/$1 with the hooks installed: the runtime directory, the configured path, and
# the three hook files, each executable. It also copies the check into the fixture's scripts directory.
make_fixture() {
	dir="$work/$1"
	mkdir -p "$dir/.husky/_" "$dir/scripts"
	git init -q "$dir"
	git -C "$dir" config --local core.hooksPath .husky/_
	for hook in pre-commit commit-msg pre-push; do
		printf '#!/usr/bin/env sh\n' >"$dir/.husky/$hook"
		chmod +x "$dir/.husky/$hook"
	done
	cp "$check" "$dir/scripts/check-hooks-installed.sh"
}

# check runs the fixture's copy of the check with no argument, so the default root is the fixture itself. It compares the
# exit status and a fragment of the output with the expected ones.
check_case() {
	name=$1
	want_status=$2
	want_text=$3
	dir=$4
	cases=$((cases + 1))
	got=0
	sh "$dir/scripts/check-hooks-installed.sh" >"$work/out" 2>&1 || got=$?
	if [ "$got" -ne "$want_status" ]; then
		printf 'FAIL %s: exit status %s, expected %s\n' "$name" "$got" "$want_status"
		failed=$((failed + 1))
		return 0
	fi
	if ! grep -Fq -- "$want_text" "$work/out"; then
		printf 'FAIL %s: output does not contain "%s"\n' "$name" "$want_text"
		failed=$((failed + 1))
		return 0
	fi
	printf 'ok %s\n' "$name"
}

make_fixture installed
check_case installed 0 'Git hooks: installed' "$work/installed"

make_fixture unset-path
git -C "$work/unset-path" config --local --unset core.hooksPath
check_case hooks-path-unset 1 'core.hooksPath is ""' "$work/unset-path"

make_fixture wrong-path
git -C "$work/wrong-path" config --local core.hooksPath .git/hooks
check_case hooks-path-not-husky 1 'expected ".husky/_"' "$work/wrong-path"

make_fixture missing-hook
rm "$work/missing-hook/.husky/commit-msg"
check_case hook-file-missing 1 'hook file .husky/commit-msg is missing' "$work/missing-hook"

make_fixture not-executable
chmod -x "$work/not-executable/.husky/pre-push"
check_case hook-file-not-executable 1 'hook file .husky/pre-push is not executable' "$work/not-executable"

printf '%s\n' "check-hooks-installed tests: $((cases - failed)) passed, $failed failed"
[ "$failed" -eq 0 ]
