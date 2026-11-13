#!/bin/sh
# Builds the release archives of retriever for one tagged version, and their SHA256SUMS. The procedure in
# docs/how-to/cut-an-alpha-pre-release.md runs it after the annotated tag exists. It builds from that tag in a clean
# clone, never from the working tree, with the Go release that go.mod names and with CGO disabled.
#
# usage: build-release.sh <version> <output-dir>
#
#   version      a v-prefixed semantic version whose annotated tag exists, such as v0.1.0
#   output-dir   a directory that is empty or does not exist yet; it receives one archive per platform and SHA256SUMS
#
# Exit statuses: 0 the assets were written, 1 a precondition or a check failed, 2 a usage error.
set -eu
export LC_ALL=C

# A Git hook in a linked worktree exports GIT_DIR, which Git prefers over -C. Without this the clone could read the
# wrong repository.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_COMMON_DIR
unset GIT_NAMESPACE GIT_PREFIX

here=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=${RELEASE_REPO:-$(git -C "$here" rev-parse --show-toplevel)}

if [ "$#" -ne 2 ]; then
	echo "usage: build-release.sh <version> <output-dir>" >&2
	exit 2
fi
version=$1
output_dir=$2

if ! printf '%s\n' "$version" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
	echo "version must be a v-prefixed semantic version, such as v0.1.0" >&2
	exit 2
fi

tag="refs/tags/$version"
if ! git -C "$repo" rev-parse -q --verify "$tag^{commit}" >/dev/null 2>&1; then
	echo "tag $version does not exist; create the annotated tag first" >&2
	exit 1
fi
if [ "$(git -C "$repo" cat-file -t "$tag")" != tag ]; then
	echo "tag $version must be annotated" >&2
	exit 1
fi

if [ -e "$output_dir" ] && [ -n "$(ls -A "$output_dir")" ]; then
	echo "output directory must be empty: $output_dir" >&2
	exit 1
fi
mkdir -p "$output_dir"
output_dir=$(CDPATH='' cd -- "$output_dir" && pwd)

work=$(mktemp -d "${TMPDIR:-/tmp}/retriever-release.XXXXXX")
cleanup() {
	rm -rf -- "$work"
}
trap cleanup EXIT HUP INT TERM

git clone --quiet --no-checkout --no-local "$repo" "$work/source"
git -C "$work/source" checkout --detach --quiet "$tag"
module="$work/source/apps/retriever-cli"

# The module names the Go release it is built with. The build uses that exact release, whatever Go the host has.
toolchain=$(sed -n 's/^toolchain[[:space:]][[:space:]]*\(go[^[:space:]]*\)[[:space:]]*$/\1/p' "$module/go.mod")
if ! printf '%s\n' "$toolchain" | grep -Eq '^go[0-9]+\.[0-9]+\.[0-9]+$'; then
	echo "go.mod must name an exact Go release in its toolchain line" >&2
	exit 1
fi
GOTOOLCHAIN=$toolchain
GOENV=off
export GOTOOLCHAIN GOENV
unset GOFLAGS GOEXPERIMENT GOAMD64 GOARM64 GOFIPS140

# The archive carries the tagged commit's time, so the archive bytes do not depend on when the build ran.
modified=$(git -C "$work/source" show -s --format=%ct "$tag^{commit}")
stamp=$(date -u -d "@$modified" +%Y%m%d%H%M.%S 2>/dev/null || date -u -r "$modified" +%Y%m%d%H%M.%S)

if tar --version 2>/dev/null | head -n 1 | grep -q '^bsdtar'; then
	bsd_tar=yes
else
	bsd_tar=no
fi

case "$(uname -s)" in
Darwin) host_os=darwin ;;
Linux) host_os=linux ;;
*) host_os= ;;
esac
case "$(uname -m)" in
arm64 | aarch64) host_arch=arm64 ;;
x86_64 | amd64) host_arch=amd64 ;;
*) host_arch= ;;
esac

# make_archive <stage-dir> <archive> writes the binary as the one member of a gzip-compressed tar. The member is owned
# by root, has mode 0755, and carries the commit's time, and gzip keeps no name or time of its own.
make_archive() {
	if [ "$bsd_tar" = yes ]; then
		COPYFILE_DISABLE=1 tar -C "$1" --format ustar --uid 0 --gid 0 --uname root --gname root \
			-cf "$work/member.tar" retriever
	else
		tar -C "$1" --format=ustar --owner=0 --group=0 --numeric-owner -cf "$work/member.tar" retriever
	fi
	gzip -n -c "$work/member.tar" >"$2"
	rm -f "$work/member.tar"
}

for target in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
	goos=${target%_*}
	goarch=${target#*_}
	stage="$work/stage-$target"
	mkdir -p "$stage"
	(
		cd "$module"
		CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -buildvcs=false \
			-o "$stage/retriever" ./cmd/retriever
	)
	chmod 0755 "$stage/retriever"
	TZ=UTC touch -t "$stamp" "$stage/retriever"

	# A binary that runs on this host must print the version and the alpha notice the tag names.
	if [ "$goos" = "$host_os" ] && [ "$goarch" = "$host_arch" ]; then
		expect=$(printf 'retriever %s\nALPHA: expect breaking changes' "${version#v}")
		got=$("$stage/retriever" --version 2>&1) || got=
		if [ "$got" != "$expect" ]; then
			echo "the host binary does not report version ${version#v}" >&2
			exit 1
		fi
	fi

	make_archive "$stage" "$output_dir/retriever_${version}_${goos}_${goarch}.tar.gz"
done

(
	cd "$output_dir"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum retriever_"${version}"_*.tar.gz >SHA256SUMS
		sha256sum -c SHA256SUMS >/dev/null
	else
		shasum -a 256 retriever_"${version}"_*.tar.gz >SHA256SUMS
		shasum -a 256 -c SHA256SUMS >/dev/null
	fi
)

echo "wrote the four platform archives and SHA256SUMS to $output_dir"
