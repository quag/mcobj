#!/bin/bash

./clean.sh
GOOS=darwin GOARCH=amd64 ./build.sh

7z a mcobj-$(git describe)-osx.7z mcobj blocks.json
