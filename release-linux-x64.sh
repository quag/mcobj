#!/bin/bash

./clean.sh
GOOS=linux GOARCH=amd64 ./build.sh

7za a mcobj-$(git describe)-linux-x64.7z mcobj blocks.json
