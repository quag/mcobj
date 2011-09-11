#!/bin/bash

BUILD=linux-x64 GOOS=linux GOARCH=amd64 $(dirname $0)/release-common.sh
