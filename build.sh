#!/bin/bash

if [ ! "$http_proxy" = "" ]
then
    docker build --no-cache --build-arg "https_proxy=$https_proxy" --build-arg "http_proxy=$http_proxy" -t pfelipefeitosa/of-watchdog:build .
else
    docker build -t pfelipefeitosa/of-watchdog:build .
fi

docker build --no-cache --build-arg PLATFORM="-darwin" -t pfelipefeitosa/of-watchdog:latest-dev-darwin . -f Dockerfile.packager
docker build --no-cache --build-arg PLATFORM="-armhf" -t pfelipefeitosa/of-watchdog:latest-dev-armhf . -f Dockerfile.packager
docker build --no-cache --build-arg PLATFORM="-arm64" -t pfelipefeitosa/of-watchdog:latest-dev-arm64 . -f Dockerfile.packager
docker build --no-cache --build-arg PLATFORM=".exe" -t pfelipefeitosa/of-watchdog:latest-dev-windows . -f Dockerfile.packager
docker build --no-cache --build-arg PLATFORM="" -t pfelipefeitosa/of-watchdog:latest-dev-x86_64 . -f Dockerfile.packager

docker create --name buildoutput pfelipefeitosa/of-watchdog:build echo

docker cp buildoutput:/go/src/github.com/paulofelipefeitosa/of-watchdog/of-watchdog ./of-watchdog
docker cp buildoutput:/go/src/github.com/paulofelipefeitosa/of-watchdog/of-watchdog-darwin ./of-watchdog-darwin
docker cp buildoutput:/go/src/github.com/paulofelipefeitosa/of-watchdog/of-watchdog-armhf ./of-watchdog-armhf
docker cp buildoutput:/go/src/github.com/paulofelipefeitosa/of-watchdog/of-watchdog-arm64 ./of-watchdog-arm64
docker cp buildoutput:/go/src/github.com/paulofelipefeitosa/of-watchdog/of-watchdog.exe ./of-watchdog.exe

docker rm buildoutput
