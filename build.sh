#!/bin/bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w' -o ./mock-pilot
docker build -t mock-pilot:latest ./
