#!/bin/bash

echo "package main" > version.go
echo >> version.go
echo "const version = \""$(git describe || echo "unknown")"\"" >> version.go

gofmt -w nbt/*.go || exit
gofmt -w *.go || exit

make -C nbt || exit
cp nbt/_obj/nbt.a nbt.a || exit

make || exit
