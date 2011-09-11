#!/bin/bash

BUILD=linux-x86 GOOS=linux GOARCH=386 $(dirname $0)/release-common.sh
