#!/bin/bash

export GOPATH=$(dirname $PWD/$0)

CMDPKG=quag.geek.nz/mcobj
CMDSRC=$GOPATH/src/$CMDPKG

echo "package main" > $CMDSRC/version.go
echo >> $CMDSRC/version.go
echo "const version = \""$(git describe || echo "unknown")"\"" >> $CMDSRC/version.go

goinstall -clean $CMDPKG || exit
