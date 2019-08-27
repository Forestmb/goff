#!/bin/sh
set -e
# To run this before every commit, use: 
#     $ ln -s "$(pwd)/build.sh" .git/hooks/pre-commit
 
dir="$(dirname "$(readlink -f "$0")")"
cd "${dir}"

export PATH="${GOPATH}/bin:${PATH}"

echo "Running go get..."
go get

echo "Running golint..."
go get golang.org/x/lint/golint
golint .

echo "Running go vet..."
go vet .

echo "Running goimports..."
go get golang.org/x/tools/cmd/goimports
goimports -w .

echo "Running go fmt..."
go fmt ./...

echo "Running tests..."
go test -v -covermode=count -coverprofile="profile.cov"
go tool cover -func profile.cov

echo "Building..."
go build .

cd debug
go build .
