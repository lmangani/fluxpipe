#!/bin/sh
export PKG_CONFIG_PATH=$(pwd)

echo "Building fluxpipe-server ..."
go build -a -ldflags '-extldflags "-static -w -ldl"' -o fluxpipe-server fluxpipe-server.go
