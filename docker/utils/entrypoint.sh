#!/usr/bin/env bash

# Wait for database
echo "Waiting for db"
./docker/utils/wait-for -t 60 db:3306
CompileDaemon -directory=. -build="go build main.go" -command=./main