#!/bin/bash

./clean.sh
GOOS=linux GOARCH=386 ./build.sh

7z a mcobj-$(git describe)-linux-x86.7z mcobj blocks.json
