#!/bin/sh
set -e
# To run this before every commit, use: 
#     $ ln -s "$(pwd)/build.sh" .git/hooks/pre-commit
 
package="github.com/Forestmb/goff"
debug="${package}/debug"

cd "${GOPATH}/src/${package}"

echo "Running go get..."
go get

echo "Running tests..."
go test -v ./...

echo "Running golint..."
go get github.com/golang/lint/golint
$GOPATH/bin/golint .

echo "Running go vet..."
go vet .

echo "Running goimports..."
go get code.google.com/p/go.tools/cmd/goimports
$GOPATH/bin/goimports -w .

echo "Running go fmt..."
go fmt ./...

echo "Building..."
go build .

cd "${GOPATH}/src/${debug}"
go build .
