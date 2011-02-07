#!/bin/bash

gofmt.exe -w *.go || exit

8g nbt.go || exit
gopack grc nbt.a nbt.8 || exit

8g -I. ntb.go mtl.go process.go sides.go sideCache.go enclosedChunk.go || exit
8l -L. -o ntb.exe ntb.8
