#!/bin/sh
# Self-test for scripts/build-release.sh. It runs the script against a fixture repository whose module names a Go
# toolchain, with a fake go command in place of the real one, so no Go build runs. Each case names the status and the
# message the script must give. The archive checks read the host's own tar and gzip, so the test runs on macOS and
# Linux alike.
set -eu

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
release="$here/build-release.sh"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cases=0
failed=0

# Fixture commits and tags take their identity from the environment of this test only. No git configuration is written.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_COMMON_DIR
unset GIT_NAMESPACE GIT_PREFIX
export GIT_AUTHOR_NAME=release-test
export GIT_AUTHOR_EMAIL=release-test@example.test
export GIT_COMMITTER_NAME=release-test
export GIT_COMMITTER_EMAIL=release-test@example.test
export GIT_AUTHOR_DATE='1700000000 +0000'
export GIT_COMMITTER_DATE='1700000000 +0000'

# The fake go command records each build's platform and flags. It then writes a shell script in place of the binary,
# which prints the version lines that FAKE_VERSION names.
bin="$work/bin"
log="$work/go.log"
mkdir -p "$bin"
cat >"$bin/go" <<'EOF'
#!/bin/sh
set -eu
out=
prev=
for arg in "$@"; do
	if [ "$prev" = -o ]; then
		out=$arg
	fi
	prev=$arg
done
printf '%s %s %s %s\n' "$GOOS" "$GOARCH" "$CGO_ENABLED" "$*" >>"$FAKE_GO_LOG"
{
	printf '#!/bin/sh\n'
	printf 'echo "retriever %s"\n' "$FAKE_VERSION"
	printf 'echo "ALPHA: expect breaking changes"\n'
} >"$out"
chmod 0755 "$out"
EOF
chmod 0755 "$bin/go"

# make_repo <name> <go.mod text> <annotated|light|none> makes the fixture repository $work/<name>. Its module lives at
# apps/retriever-cli, as the real one does, and the tag v0.1.0 is made the way the test names.
make_repo() {
	repo="$work/$1"
	mkdir -p "$repo/apps/retriever-cli/cmd/retriever"
	printf '%b' "$2" >"$repo/apps/retriever-cli/go.mod"
	printf 'package main\n\nfunc main() {}\n' >"$repo/apps/retriever-cli/cmd/retriever/main.go"
	git -C "$repo" init -q
	git -C "$repo" add -A
	git -C "$repo" commit -q -m "chore: fixture"
	case "$3" in
	annotated) git -C "$repo" tag -a v0.1.0 -m "v0.1.0" ;;
	light) git -C "$repo" tag v0.1.0 ;;
	esac
}

# release_run <repo> <arguments...> runs the script against the fixture repository, with the fake go command first on
# PATH.
release_run() {
	repo=$1
	shift
	env RELEASE_REPO="$repo" FAKE_GO_LOG="$log" FAKE_VERSION="$fake_version" PATH="$bin:$PATH" sh "$release" "$@"
}

# check <name> <want-status> <want-text> <command...> runs the command and compares its status and output with the
# expected ones.
check() {
	name=$1
	want=$2
	text=$3
	shift 3
	cases=$((cases + 1))
	got=0
	"$@" >"$work/out" 2>&1 || got=$?
	if [ "$got" -ne "$want" ] || ! grep -F -q -- "$text" "$work/out"; then
		failed=$((failed + 1))
		echo "FAIL $name: want status $want with '$text', got status $got:" >&2
		sed 's/^/  /' "$work/out" >&2
	else
		echo "ok   $name"
	fi
}

# assert <name> <want> <got> compares two values.
assert() {
	cases=$((cases + 1))
	if [ "$2" = "$3" ]; then
		echo "ok   $1"
	else
		failed=$((failed + 1))
		echo "FAIL $1: want '$2', got '$3'" >&2
	fi
}

# verify_sums <dir> checks the archives in a directory against its SHA256SUMS, with the tool the host has.
verify_sums() {
	(
		cd "$1"
		if command -v sha256sum >/dev/null 2>&1; then
			sha256sum -c SHA256SUMS
		else
			shasum -a 256 -c SHA256SUMS
		fi
	) >/dev/null 2>&1
}

module_ok='module example.test/fixture\n\ngo 1.27.0\n\ntoolchain go1.27.4\n'
make_repo tagged "$module_ok" annotated
make_repo light "$module_ok" light
make_repo untagged "$module_ok" none
make_repo no-toolchain 'module example.test/fixture\n\ngo 1.27.0\n' annotated
tagged=$work/tagged
fake_version=0.1.0

check "no arguments is a usage error" 2 "usage: build-release.sh" release_run "$tagged"
check "a version without the v prefix is refused" 2 "version must be a v-prefixed semantic version" \
	release_run "$tagged" 0.1.0 "$work/refused-version"
check "a missing tag is refused" 1 "tag v0.1.0 does not exist" \
	release_run "$work/untagged" v0.1.0 "$work/refused-missing"
check "a lightweight tag is refused" 1 "tag v0.1.0 must be annotated" \
	release_run "$work/light" v0.1.0 "$work/refused-light"
check "a go.mod without a toolchain line is refused" 1 "toolchain line" \
	release_run "$work/no-toolchain" v0.1.0 "$work/refused-toolchain"

mkdir -p "$work/not-empty"
printf 'stale\n' >"$work/not-empty/stale.txt"
check "a non-empty output directory is refused" 1 "output directory must be empty" \
	release_run "$tagged" v0.1.0 "$work/not-empty"

fake_version=0.2.0
check "a host binary that reports another version is refused" 1 "does not report version 0.1.0" \
	release_run "$tagged" v0.1.0 "$work/refused-version-report"
fake_version=0.1.0

first=$work/first
: >"$log"
check "a tagged version builds the four platform archives" 0 "wrote the four platform archives" \
	release_run "$tagged" v0.1.0 "$first"

expected_names='SHA256SUMS
retriever_v0.1.0_darwin_amd64.tar.gz
retriever_v0.1.0_darwin_arm64.tar.gz
retriever_v0.1.0_linux_amd64.tar.gz
retriever_v0.1.0_linux_arm64.tar.gz'
# shellcheck disable=SC2012 # the listed names are fixed archive names, so ls is safe here.
assert "the output holds exactly the four archives and SHA256SUMS" "$expected_names" "$(LC_ALL=C ls -A "$first" | LC_ALL=C sort)"

if verify_sums "$first"; then
	assert "SHA256SUMS verifies every archive" "ok" "ok"
else
	assert "SHA256SUMS verifies every archive" "ok" "failed"
fi
assert "SHA256SUMS lists one line per archive" "4" "$(wc -l <"$first/SHA256SUMS" | tr -d ' ')"

assert "each platform builds with CGO disabled, trimmed paths, and no VCS stamp" "4" \
	"$(grep -c -- '^[a-z]* [a-z0-9]* 0 build -trimpath -buildvcs=false -o ' "$log")"

for target in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
	archive=$first/retriever_v0.1.0_$target.tar.gz
	goos=${target%_*}
	goarch=${target#*_}
	listed=$(tar -tzf "$archive")
	assert "$target archive holds only the retriever binary" "retriever" "$listed"
	mode=$(tar -tvzf "$archive" | cut -c1-10)
	assert "$target archive member is mode 0755" "-rwxr-xr-x" "$mode"
	assert "$target archive member is not owned by the build user" "0" \
		"$(tar -tvzf "$archive" | grep -c -- "$(id -un)" || true)"
	assert "$target build targets its platform" "1" \
		"$(grep -c -- "^$goos $goarch 0 " "$log")"
done

# A second build, a second later, must reproduce every byte: the archive carries the release commit's time, not the
# build's time.
sleep 1
: >"$log"
second=$work/second
release_run "$tagged" v0.1.0 "$second" >/dev/null 2>&1
for path in "$first"/*; do
	name=${path##*/}
	if cmp -s "$first/$name" "$second/$name"; then
		assert "rebuild reproduces $name byte for byte" "same" "same"
	else
		assert "rebuild reproduces $name byte for byte" "same" "different"
	fi
done

if [ "$failed" -ne 0 ]; then
	echo "$failed of $cases build-release cases failed" >&2
	exit 1
fi
echo "all $cases build-release cases passed"
