#!/bin/bash

# Run update-version.sh before building the release
VERSION=$(sed -n '3s/.*"\([^"]*\)".*/\1/p' cmd/mcobj/version.go)

BUILD=$(mktemp --tmpdir -d mcobj-buildXXXXXXXXX)
echo $VERSION $BUILD

mkdir $BUILD/windows
cp blocks.json $BUILD/windows
cp LICENSE $BUILD/windows/LICENSE.txt
(
    cd cmd/mcobj
    GOOS=windows GOARCH=386 go build -o $BUILD/windows/mcobj.exe
    cd $BUILD/windows
    7z a $BUILD/mcobj-$VERSION-windows.7z mcobj.exe blocks.json LICENSE.txt
)

mkdir $BUILD/osx
cp blocks.json $BUILD/osx
cp LICENSE $BUILD/osx/LICENSE
(
    cd cmd/mcobj
    GOOS=darwin GOARCH=amd64 go build -o $BUILD/osx/mcobj
    cd $BUILD/osx
    7z a $BUILD/mcobj-$VERSION-osx.7z mcobj blocks.json LICENSE
)

mkdir $BUILD/linux-x86
cp blocks.json $BUILD/linux-x86
cp LICENSE $BUILD/linux-x86/LICENSE
(
    cd cmd/mcobj
    GOOS=linux GOARCH=386 go build -o $BUILD/linux-x86/mcobj
    cd $BUILD/linux-x86
    7z a $BUILD/mcobj-$VERSION-linux-x86.7z mcobj blocks.json LICENSE
)

mkdir $BUILD/linux-x64
cp blocks.json $BUILD/linux-x64
cp LICENSE $BUILD/linux-x64/LICENSE
(
    cd cmd/mcobj
    GOOS=linux GOARCH=amd64 go build -o $BUILD/linux-x64/mcobj
    cd $BUILD/linux-x64
    7z a $BUILD/mcobj-$VERSION-linux-x64.7z mcobj blocks.json LICENSE
)

mv $BUILD $VERSION
