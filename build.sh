#!/bin/bash

echo "package main" > version.go
echo >> version.go
echo "const version = \""$(git describe || echo "unknown")"\"" >> version.go

gofmt.exe -w *.go || exit

8g nbt.go || exit
gopack grc nbt.a nbt.8 || exit

8g -I. mcobj.go version.go mtl.go process.go sides.go sideCache.go enclosedChunk.go || exit
8l -L. -o mcobj.exe mcobj.8
