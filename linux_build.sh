#!/bin/bash

#CROSS-COMPILE: CREATE LINUX EXECUTABLE FROM MAC

#chmod +x /Developer/Projects/Go/SMS/linux_build.sh 
cd `dirname $0`
GOOS=linux GOARCH=amd64 go build -o /Developer/Vagrant_Go/main_linux *.go