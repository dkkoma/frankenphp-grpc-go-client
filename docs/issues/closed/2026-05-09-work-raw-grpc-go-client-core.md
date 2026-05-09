# Raw grpc-go Client Core

## State

closed

## Source

work

## Context

Implement the first Go-side core that can send and receive serialized protobuf
bytes through grpc-go without generated protobuf types. This is the foundation
for the FrankenPHP bridge.

## Acceptance Criteria

- Go module path is `github.com/dkkoma/frankenphp-grpc-go-client`.
- Unary calls support raw bytes, metadata, status, trailers, and deadlines.
- Server streaming calls support raw bytes, initial metadata before reads,
  final status, trailers, and non-OK final status after messages.
- Non-status native/transport failures remain Go errors for the future PHP
  bridge to map to exceptions.
- Channel close is idempotent and blocks new calls.
- Tests run against an in-process grpc-go service.

## Verification

```sh
docker compose run --rm dev go test ./...
```

Passed.

```sh
docker compose run --rm dev go test -race ./...
```

Passed.
