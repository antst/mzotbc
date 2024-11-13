FROM golang:1.23-alpine as build
RUN apk add --update --no-cache make gcc binutils-gold musl-dev git

WORKDIR /project
ENV GOPATH /go
ENV GOCACHE /go-cache
# Seperate step to allow docker layer caching
COPY go.* ./
RUN --mount=type=cache,target=/go-cache --mount=type=cache,target=/go go mod download

#COPY depsbuild ./depsbuild
#COPY Makefile ./
#RUN --mount=type=cache,target=/go-cache --mount=type=cache,target=/go make deps

COPY . ./

RUN --mount=type=cache,target=/go-cache --mount=type=cache,target=/go make build


FROM alpine:latest as production-build

RUN mkdir -p /opt/bin
COPY --from=build /project/bin/mzotbc /opt/bin/mzotbc

RUN mkdir -p /data
COPY --from=build /project/example_config.yaml /data/config.yaml

VOLUME ["/data"]

# This command runs your application, comment out this line to compile only
CMD ["/opt/bin/mzotbc","-c","/data/config.yaml","-d","/data/mzotbc.db"]

LABEL Name=pengine Version=0.0.1

