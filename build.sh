#!/bin/sh

echo "=== Building mac amd64 ==="
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o ./dist/pdf-annots2json.darwin.amd64 .

echo "=== Building mac arm64 ==="
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o ./dist/pdf-annots2json.darwin.arm64 .

# https://github.com/messense/homebrew-macos-cross-toolchains
echo "=== Building linux amd64 ==="
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC="x86_64-unknown-linux-gnu-gcc" go build -o ./dist/pdf-annots2json.linux.amd64 .

# https://words.filippo.io/easy-windows-and-linux-cross-compilers-for-macos/
echo "=== Building windows amd64 ==="
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="x86_64-w64-mingw32-gcc" go build -o ./dist/pdf-annots2json.windows.amd64.exe .