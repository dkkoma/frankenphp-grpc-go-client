FROM golang:1.26-bookworm AS go
FROM composer:2 AS composer
FROM php:8.4-cli-bookworm

ENV DEBIAN_FRONTEND=noninteractive
ENV PATH="/usr/local/go/bin:${PATH}"

COPY --from=go /usr/local/go /usr/local/go
COPY --from=composer /usr/bin/composer /usr/bin/composer

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        curl \
        git \
        libargon2-dev \
        libcurl4-openssl-dev \
        libncurses-dev \
        libonig-dev \
        libreadline-dev \
        libsqlite3-dev \
        libssl-dev \
        libxml2-dev \
        pkg-config \
        unzip \
        zlib1g-dev \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt \
    && rm -rf /var/lib/apt/lists/*

RUN go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest \
    && ln -s /root/go/bin/xcaddy /usr/local/bin/xcaddy

WORKDIR /workspace
