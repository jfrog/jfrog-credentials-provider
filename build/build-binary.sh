#!/usr/bin/env bash

# Script inspired by https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04

errorExit () {
    echo; echo "ERROR: $1"; echo
    exit 1
}

read -r -a PLATFORMS <<< "$PLATFORMS"
echo "PLATFORMS: $PLATFORMS"
echo "BUILD_DIR: $BUILD_DIR"
echo "BIN: $BIN"

rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

#PLATFORMS=("darwin/amd64" "linux/arm64" "linux/amd64" "windows/amd64" "windows/386")
for p in "${PLATFORMS[@]}"; do
    platform_array=(${p//\// })
    GOOS=${platform_array[0]}
    GOARCH=${platform_array[1]}

    echo -e "\nBuilding"
    echo "OS:   $GOOS"
    echo "ARCH: $GOARCH"
    final_name=$BIN'-'$GOOS'-'$GOARCH
    if [ "$GOOS" = "windows" ]; then
        final_name+='.exe'
    fi

    env GOOS="$GOOS" GOARCH="$GOARCH" go build -o $BUILD_DIR/$final_name ../ || errorExit "Building $final_name failed"
done

echo -e "\nDone!\nThe following binaries were created in the bin/ directory:"
ls -1 $BUILD_DIR/
echo