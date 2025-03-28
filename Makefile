# Copyright (c)
BUILD_VERSION ?= $(shell git describe --tags --dirty)
DOCKER_IMAGE  ?= mzotbc
DOCKER_TAG    ?= develop

all: build

clean:
	rm -rf ./build ./bin

build:
	CGO_ENABLED=1 go build -ldflags "${LDFLAGS} -X main.version=${BUILD_VERSION}" -buildvcs=false -trimpath  -o ./bin/mzotbc ./mzotbc/

run: build
	./bin/mzotbc

db:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate

docker-build: DOCKER_BUILDKIT=1
docker-build:
	docker build --build-arg BUILD_VERSION=${BUILD_VERSION} -t ${DOCKER_IMAGE}:${DOCKER_TAG} .

docker-run: docker-build
	docker run  ${DOCKER_IMAGE}:${DOCKER_TAG}
