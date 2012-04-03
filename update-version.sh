#!/bin/bash

(
    echo "package main"
    echo
    echo "const version = \""$(git describe || echo "unknown")"\""
) > cmd/mcobj/version.go
