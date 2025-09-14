#!/bin/bash

OUTPUT_DIR="release"
APP_ENTRY="main.go"
APP_NAME="go_sqlite_web"
VERSION=\"$1\"

mkdir -p $OUTPUT_DIR

# 构建 Linux
GOOS=linux GOARCH=amd64 go build -o ${OUTPUT_DIR}/${APP_NAME}_linux_amd64 \
    -ldflags "-s -w \
    -X 'main.version=${VERSION}' \
    -X 'main.date=$(date +%Y-%m-%d.%H:%M)'" \
    ${APP_ENTRY}

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o ${OUTPUT_DIR}/${APP_NAME}_linux_arm64 \
    -ldflags "-s -w \
    -X 'main.version=${VERSION}' \
    -X 'main.date=$(date +%Y-%m-%d.%H:%M)'" \
    ${APP_ENTRY}

# 构建 macOS m系列
GOOS=darwin GOARCH=arm64 go build -o ${OUTPUT_DIR}/${APP_NAME}_darwin_arm64 \
    -ldflags "-s -w \
    -X 'main.version=${VERSION}' \
    -X 'main.date=$(date +%Y-%m-%d.%H:%M)'" \
    ${APP_ENTRY}

# 构建 macOS intel
GOOS=darwin GOARCH=amd64 go build -o ${OUTPUT_DIR}/${APP_NAME}_darwin_amd64 \
    -ldflags "-s -w \
    -X 'main.version=${VERSION}' \
    -X 'main.date=$(date +%Y-%m-%d.%H:%M)'" \
    ${APP_ENTRY}

# 构建 Windows amd64
GOOS=windows GOARCH=amd64 go build -o ${OUTPUT_DIR}/${APP_NAME}_windows_amd64.exe \
    -ldflags "-s -w \
    -X 'main.version=${VERSION}' \
    -X 'main.date=$(date +%Y-%m-%d.%H:%M)'" \
    ${APP_ENTRY}

# 构建 Windows arm64
GOOS=windows GOARCH=arm64 go build -o ${OUTPUT_DIR}/${APP_NAME}_windows_arm64.exe \
    -ldflags "-s -w \
    -X 'main.version=${VERSION}' \
    -X 'main.date=$(date +%Y-%m-%d.%H:%M)'" \
    ${APP_ENTRY}


echo "Build completed. Binaries are in the '${OUTPUT_DIR}' directory."
