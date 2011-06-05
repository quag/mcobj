#!/bin/bash

BUILD=windows GOOS=windows GOARCH=386 $(dirname $0)/release-common.sh
