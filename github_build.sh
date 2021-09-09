#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

export VERSION="${GITHUB_REF##*/}-${GITHUB_SHA}"

echo "VERSION: ${VERSION}"

echo "checking go fmt ./..."
gofmt=$(go fmt ./...)
if [[ ${gofmt} ]]; then
    echo "the following files need go fmt:"
    echo "${gofmt}"
    echo "run `go fmt ./...`"
    exit 1
fi

echo "fetching dependencies"
go mod download

echo "testing"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -v ./...

echo "building service"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/siemens/link-checker-service/infrastructure.Version=$VERSION" -o link-checker-service-lin .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/siemens/link-checker-service/infrastructure.Version=$VERSION" -o link-checker-service-win.exe .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/siemens/link-checker-service/infrastructure.Version=$VERSION" -o link-checker-service-osx .

./link-checker-service-lin version
./link-checker-service-lin || true
./link-checker-service-lin help serve || true

echo "building sample UI"
cd $GITHUB_WORKSPACE/test/jquery_example && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $GITHUB_WORKSPACE/link-checker-example-ui-lin .
cd $GITHUB_WORKSPACE/test/jquery_example && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $GITHUB_WORKSPACE/link-checker-example-ui-win.exe .
cd $GITHUB_WORKSPACE/test/jquery_example && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $GITHUB_WORKSPACE/link-checker-example-ui-osx .
cp $GITHUB_WORKSPACE/test/jquery_example/start_example.bat $GITHUB_WORKSPACE/

cd $GITHUB_WORKSPACE

echo "archiving"

mv link-checker-example-ui-win.exe link-checker-example-ui.exe
mv link-checker-service-win.exe link-checker-service.exe
zip link-checker-service-win.zip *.exe README.md start_example.bat .link-checker-service.toml

mv link-checker-example-ui-lin link-checker-example-ui
mv link-checker-service-lin link-checker-service
tar cvzf link-checker-service-lin.tgz link-checker-service link-checker-example-ui README.md .link-checker-service.toml

mv link-checker-example-ui-osx link-checker-example-ui
mv link-checker-service-osx link-checker-service
tar cvzf link-checker-service-osx.tgz link-checker-service link-checker-example-ui README.md .link-checker-service.toml

ls -rtl
