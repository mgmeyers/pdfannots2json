#!/bin/sh

rm -rf ./dist
mkdir ./dist

echo "=== Building mac amd64 ==="
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -trimpath -o ./dist/pdfannots2json .
cd ./dist
tar -czvf "./pdfannots2json.Mac.Intel.tar.gz" ./pdfannots2json
rm ./pdfannots2json
cd ..

echo "=== Building mac arm64 ==="
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -trimpath -o ./dist/pdfannots2json .
cd ./dist
tar -czvf "./pdfannots2json.Mac.M1.tar.gz" ./pdfannots2json
rm ./pdfannots2json
cd ..

# https://github.com/messense/homebrew-macos-cross-toolchains
echo "=== Building linux amd64 ==="
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC="x86_64-unknown-linux-gnu-gcc" go build -trimpath -o ./dist/pdfannots2json .
cd ./dist
tar -czvf "./pdfannots2json.Linux.x64.tar.gz" ./pdfannots2json
rm ./pdfannots2json
cd ..

# https://words.filippo.io/easy-windows-and-linux-cross-compilers-for-macos/
echo "=== Building windows amd64 ==="
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="x86_64-w64-mingw32-gcc" go build -trimpath -o ./dist/pdfannots2json.exe .
cd ./dist
zip "./pdfannots2json.Windows.x64.zip" ./pdfannots2json.exe
rm ./pdfannots2json.exe
cd ..
