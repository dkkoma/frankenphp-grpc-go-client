EXTENSION_TAGS := frankengrpc_extension,nowatcher,nobadger,nomysql,nopgx
FRANKENPHP_BUILD_TAGS := frankengrpc_extension,nowatcher,nobadger,nomysql,nopgx,nomercure,nobrotli
PHP_CFLAGS := -D_GNU_SOURCE $(shell php-config --includes)
PHP_LDFLAGS := $(shell php-config --ldflags) $(shell php-config --libs)

.PHONY: test
test:
	go test ./...

.PHONY: test-race
test-race:
	go test -race ./...

.PHONY: lint-stubs
lint-stubs:
	php -l stubs/FrankenGrpc.php
	php -l bridge/frankengrpc.stub.php

.PHONY: test-extension
test-extension:
	CGO_CFLAGS="$(PHP_CFLAGS)" CGO_LDFLAGS="$(PHP_LDFLAGS)" go test -tags=$(EXTENSION_TAGS) .

.PHONY: verify
verify: test test-race lint-stubs test-extension

.PHONY: build-frankenphp
build-frankenphp:
	mkdir -p bin
	CGO_ENABLED=1 \
	XCADDY_GO_BUILD_FLAGS="-tags=$(FRANKENPHP_BUILD_TAGS)" \
	CGO_CFLAGS="$(PHP_CFLAGS)" \
	CGO_LDFLAGS="$(PHP_LDFLAGS)" \
	xcaddy build \
		--output bin/frankenphp \
		--with github.com/dunglas/frankenphp/caddy \
		--with github.com/dkkoma/frankenphp-grpc-go-client=.

.PHONY: smoke-api
smoke-api: build-frankenphp
	./bin/frankenphp php-cli tests/smoke/api.php
