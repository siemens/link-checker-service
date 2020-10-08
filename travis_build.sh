#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

export VERSION="${TRAVIS_BRANCH}-${TRAVIS_COMMIT}"

echo "VERSION: ${VERSION}"

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
cd $TRAVIS_BUILD_DIR/test/jquery_example && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $TRAVIS_BUILD_DIR/link-checker-example-ui-lin .
cd $TRAVIS_BUILD_DIR/test/jquery_example && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $TRAVIS_BUILD_DIR/link-checker-example-ui-win.exe .
cd $TRAVIS_BUILD_DIR/test/jquery_example && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $TRAVIS_BUILD_DIR/link-checker-example-ui-osx .
cp $TRAVIS_BUILD_DIR/test/jquery_example/start_example.bat $TRAVIS_BUILD_DIR/

cd $TRAVIS_BUILD_DIR

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
