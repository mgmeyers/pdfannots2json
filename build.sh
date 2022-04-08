#!/bin/sh

rm -rf ./dist
mkdir ./dist

echo "=== Building mac amd64 ==="
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -trimpath -o ./dist/pdf-annots2json .
cd ./dist
tar -czvf ./pdf-annots2json.mac.intel.tar.gz ./pdf-annots2json
rm ./pdf-annots2json
cd ..

echo "=== Building mac arm64 ==="
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -trimpath -o ./dist/pdf-annots2json .
cd ./dist
tar -czvf ./pdf-annots2json.mac.m1.tar.gz ./pdf-annots2json
rm ./pdf-annots2json
cd ..

# https://github.com/messense/homebrew-macos-cross-toolchains
echo "=== Building linux amd64 ==="
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC="x86_64-unknown-linux-gnu-gcc" go build -trimpath -o ./dist/pdf-annots2json .
cd ./dist
tar -czvf ./pdf-annots2json.linux.x64.tar.gz ./pdf-annots2json
rm ./pdf-annots2json
cd ..

# https://words.filippo.io/easy-windows-and-linux-cross-compilers-for-macos/
echo "=== Building windows amd64 ==="
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="x86_64-w64-mingw32-gcc" go build -trimpath -o ./dist/pdf-annots2json.exe .
cd ./dist
zip ./pdf-annots2json.windows.x64.zip ./pdf-annots2json.exe
rm ./pdf-annots2json.exe
cd ..
