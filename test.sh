#!/bin/bash

set -e

# print a command and execute it
show() {
	echo "$@" >&2
	eval "$@"
}

fatal() {
	echo "$@" >&2
	exit 1
}

GOFILES=$(find * -name '*.go' -not -path 'vendor/*' -not -name 'bindata.go')

echo "Formatting checks..."

FMT_FILES="$(gofmt -s -l $GOFILES)"
if [[ -n $FMT_FILES ]]; then
	fatal "Run 'gofmt -s -w' on these files:\n$FMT_FILES"
fi

echo "gofmt check is ok!"

IMP_FILES="$(goimports -l $GOFILES)"
if [[ -n $IMP_FILES ]]; then
	fatal "Run 'goimports -w' on these files:\n$IMP_FILES"
fi

echo "goimports check is ok!"

for pkg in $(go list f-license/...);
do
    echo "Testing... $pkg"
    go test -race -v $pkg
done
