PROJECT_NAME=embroidery-buddy
BUILD_DIR=build/bin
GO_FLAGS=-ldflags="-s -w"
GOARCH_RPI=arm
GOARM_RPI=6
GOOS_RPI=linux
BINARY_NAME=embroidery-usbd

.PHONY: all build build-rpi clean test run

all: clean test build build-rpi

build:
	go build ${GO_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME} cmd/embroidery-usbd/main.go

build-rpi:
	GOOS=${GOOS_RPI} GOARCH=${GOARCH_RPI} GOARM=${GOARM_RPI} \
		go build ${GO_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm cmd/embroidery-usbd/main.go

build-all: build build-rpi

clean:
	go clean
	rm -rf ${BUILD_DIR}

test:
	go test -v ./...
