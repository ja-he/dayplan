#!/bin/env sh

# usage:
#   build the binary in working directory:
#     $ ./build.sh
#   install the binary to $GOPATH or $HOME/go/bin
#     $ ./build.sh install

project_base="github.com/ja-he/dayplan"
source_root="${project_base}/src"

# get the version tag that points to HEAD
version="$(git tag --points-at HEAD | grep '^v[0-9]\+.[0-9]\+.[0-9]\+$')"
if [ -z "${version}" ]; then
  version="untagged"
fi

# get the hash for HEAD, with '-dirty' appended if changes are made.
hash="$(git rev-parse --short HEAD)"
if [ "$(git diff --name-only | wc -l)" -gt "0" ]; then
  hash="${hash}-dirty"
fi

# build / install (based on arg)
case ${1} in
  install)
    printf "installing version '%s' at '%s'...\n" "${version}" "${hash}"
    go install -ldflags="-X '${source_root}/control/cli.version=${version}' -X '${source_root}/control/cli.hash=${hash}'"
    printf "done\n"
    ;;
  "")
    printf "building version '%s' at '%s'...\n" "${version}" "${hash}"
    go build -ldflags="-X '${source_root}/control/cli.version=${version}' -X '${source_root}/control/cli.hash=${hash}'"
    printf "done\n"
    ;;
  *)
    printf 'unknown command '%s', aborting...' "${1}"
    exit 1
    ;;
esac
