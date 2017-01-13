#!/bin/bash

go-bindata -o cmd/fbconvert/bindata.go ./data/
go install -v -ldflags "-X main.VERSION=`cat ./VERSION`" ./cmd/...
