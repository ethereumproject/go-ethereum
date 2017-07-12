#!/bin/sh
set -e

TAR_FILE="/tmp/goreleaser.tar.gz"
RELEASES_URL="https://github.com/goreleaser/goreleaser/releases"
test -z "$TMPDIR" && TMPDIR="$(mktemp -d)"

last_version() {
  curl -sL -o /dev/null -w %{url_effective} "$RELEASES_URL/latest" |
    rev |
    cut -f1 -d'/'|
    rev
}

download() {
  test -z "$VERSION" && VERSION="$(last_version)"
  test -z "$VERSION" && {
    echo "Unable to get goreleaser version." >&2
    exit 1
  }
  rm -f "$TAR_FILE"
  curl -s -L -o "$TAR_FILE" \
    "$RELEASES_URL/download/$VERSION/goreleaser_$(uname -s)_$(uname -m).tar.gz"
}

download
tar -xf "$TAR_FILE" -C "$TMPDIR"

# This is the modification which requires a local copy of this install file --
# we want to use `--skip-publish` and `--skip-validate` options. It would be possible to
# modify the curl-ed file before running it in a one-liner, but awkward.
# This way we get transparency, too.
# --skip-publish turns off publishing releases to github (we're using custom builds site)
# --skip-validate turns off validating git cleanliness (we'll have test builds and janus dirtying things), and allows not-on-a-tag builds
"${TMPDIR}"/goreleaser --skip-publish --skip-validate
