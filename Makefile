PROJECT_NAME=embroidery-buddy
BUILD_DIR=build/bin
GO_FLAGS=-ldflags="-s -w"
GOARCH_RPI=arm
GOARM_RPI=6
GOOS_RPI=linux
BINARY_NAME=embroidery-usbd
BENCHMARK_NAME=benchmark-copy

.PHONY: all build build-rpi build-benchmark build-benchmark-rpi build-all clean test run benchmark

all: clean test build-all

build:
	go build ${GO_FLAGS} -o ${BUILD_DIR}/${BINARY_NAME} cmd/embroidery-usbd/main.go
	go build ${GO_FLAGS} -o ${BUILD_DIR}/${BENCHMARK_NAME} cmd/benchmark-copy/main.go

build-rpi:
	GOOS=${GOOS_RPI} GOARCH=${GOARCH_RPI} GOARM=${GOARM_RPI} \
		go build ${GO_FLAGS} -o ${BUILD_DIR}/linux-arm/${BINARY_NAME}-linux-arm cmd/embroidery-usbd/main.go
	GOOS=${GOOS_RPI} GOARCH=${GOARCH_RPI} GOARM=${GOARM_RPI} \
		go build ${GO_FLAGS} -o ${BUILD_DIR}/linux-arm/${BENCHMARK_NAME}-linux-arm cmd/benchmark-copy/main.go

build-all: build build-rpi

build-benchmark:
	go build ${GO_FLAGS} -o ${BUILD_DIR}/${BENCHMARK_NAME} cmd/benchmark-copy/main.go

build-benchmark-rpi:
	GOOS=${GOOS_RPI} GOARCH=${GOARCH_RPI} GOARM=${GOARM_RPI} \
		go build ${GO_FLAGS} -o ${BUILD_DIR}/${BENCHMARK_NAME}-linux-arm cmd/benchmark-copy/main.go

copy: build-rpi
	rsync build/bin/linux-arm/* dietpi@dietpi.local:/opt/embroiderybuddy/bin/

clean:
	go clean
	rm -rf ${BUILD_DIR}

test:
	go test -v ./...

benchmark: build
	@./scripts/run-benchmark.sh
