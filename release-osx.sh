#!/bin/bash

BUILD=osx GOOS=darwin GOARCH=amd64 $(dirname $0)/release-common.sh
