# frankenphp-grpc-go-client

FrankenPHP PHP extension prototype for calling gRPC services through `grpc-go`.

The target PHP API is the `FrankenGrpc` surface consumed by
`github.com/dkkoma/php-grpc-lite-gax`. The gRPC runtime is `grpc-go`, not gRPC
Core. CGo is used for the PHP/Zend native extension boundary.

## Current Status

- Go module path: `github.com/dkkoma/frankenphp-grpc-go-client`
- Implemented: raw-byte grpc-go unary and server-streaming client core
- Implemented: in-process grpc-go tests for payload, metadata, status,
  deadline, stream lifecycle, and channel close behavior
- Implemented: manual CGo/Zend bridge exposing the `FrankenGrpc` PHP classes
- Implemented: FrankenPHP binary build and PHP API smoke
- Planned next: Spanner emulator smoke through the consumer repository's
  FrankenPHP transport path

## Development

All build and test commands are expected to run in the dev container:

```sh
docker compose run --rm dev make test
```

Race testing:

```sh
docker compose run --rm dev make test-race
```

PHP stub lint:

```sh
docker compose run --rm dev make lint-stubs
```

FrankenPHP bridge compile check, once native FrankenPHP build dependencies are
available:

```sh
docker compose run --rm dev make test-extension
```

Full verification:

```sh
docker compose run --rm dev make verify
```

Build a FrankenPHP binary with this extension:

```sh
docker compose run --rm dev make build-frankenphp
```

The container includes Go, PHP 8.4 CLI/development tooling, Composer, and C build
tooling. FrankenPHP-specific build tooling will be added when the native bridge
scaffold lands.
