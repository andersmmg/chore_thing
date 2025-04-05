#!/bin/bash
GOOS=windows GOARCH=amd64 go build -o bin/chore_thing-amd64.exe -ldflags "-H=windowsgui"
GOOS=linux GOARCH=amd64 go build -o bin/chore_thing-amd64-linux
