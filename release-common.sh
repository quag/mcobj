#!/bin/bash

export GOPATH=$(dirname $PWD/$0)

CMDPKG=quag.geek.nz/mcobj
CMDSRC="$GOPATH/src/$CMDPKG"

echo "package main" > "$CMDSRC/version.go"
echo >> "$CMDSRC/version.go"
echo "const version = \""$(git describe || echo "unknown")"\"" >> "$CMDSRC/version.go"

CMDPATH="$GOPATH/cmd/${GOOS}_$GOARCH"

rm -f "$GOPATH/bin/"* || exit
rm -f "$CMDPATH/"* || exit

goinstall -clean "$CMDPKG" || exit

mkdir -p "$CMDPATH" || exit
cp "$GOPATH/bin/"* "$CMDPATH/" || exit
cp "$CMDSRC/blocks.json" "$CMDPATH/" || exit

(
    cd "$CMDPATH" || exit
    7z a ../../mcobj-$(git describe)-$BUILD.7z "*" || exit
)
