#!/bin/sh

docker run -i -v `pwd`:/deleted_mapping_exporter alpine:edge /bin/sh << 'EOF'
set -ex

# Install prerequisites for the build process.
apk update
apk add ca-certificates git go libc-dev
update-ca-certificates

# Build the phpfpm_exporter.
cd /deleted_mapping_exporter
export GOPATH=/gopath
go get -d ./...
go build --ldflags '-extldflags "-static"'
strip deleted_mapping_exporter
EOF