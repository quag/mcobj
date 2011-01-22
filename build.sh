#!/bin/bash

gofmt.exe -w *.go
8g ntb.go mtl.go process.go && 8l -o ntb.exe ntb.8
