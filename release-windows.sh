#!/bin/bash

./clean.sh
GOOS=windows GOARCH=386 ./build.sh

7z a mcobj-$(git describe)-windows.7z mcobj.exe blocks.json
