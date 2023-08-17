# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.15-alpine AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates git
RUN apk add build-base

WORKDIR /src

RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
# restore dependencies
# COPY go.mod go.sum ./
COPY go.mod ./
RUN go mod download
COPY . .

# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /productcatalogservice .

FROM alpine AS release
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates procps curl
RUN mkdir /lib64
RUN ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

RUN GRPC_HEALTH_PROBE_VERSION=v0.3.6 && \
    wget -qO/bin/grpc_health_probe https://hub.fastgit.xyz/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe
WORKDIR /src
COPY products.json .
COPY fault.yaml .
COPY log.csv .
COPY --from=builder /productcatalogservice ./server

# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single
ENV GO_FAILPOINTS_HTTP="127.0.0.1:12346"

EXPOSE 3550
ENTRYPOINT ["/src/server"]

