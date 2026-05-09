# frankenphp-grpc-go-client

FrankenPHP PHP extension prototype for calling gRPC services through `grpc-go`.

The target PHP API is the `FrankenGrpc` surface consumed by
`github.com/dkkoma/php-grpc-lite-gax`. The gRPC runtime is `grpc-go`, not gRPC
Core. CGo is used for the PHP/Zend native extension boundary.

## Current Status

- Go module path: `github.com/dkkoma/frankenphp-grpc-go-client`
- Implemented: raw-byte grpc-go unary and server-streaming client core
- Implemented: in-process grpc-go tests for payload, metadata, status,
  deadline, receive-size limits, stream lifecycle, and channel close behavior
- Implemented: manual CGo/Zend bridge exposing the `FrankenGrpc` PHP classes
- Implemented: PHP `Channel` option parsing for the php-grpc-lite transport
  handoff, including tolerated credentials objects, authority override, user
  agent placeholder, receive-size limit, and metadata-size dial limits
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

## Channel Options

`FrankenGrpc\Channel::__construct(string $target, array $options = [])` accepts
the original `Grpc\Channel` option array from `php-grpc-lite`. Unknown options
are ignored for compatibility.

| Option | Current behavior |
| --- | --- |
| `credentials` | Accepted and retained as a compatibility signal, but plaintext transport is still used in this prototype. |
| `grpc.default_authority` | Applied via grpc-go authority override. |
| `grpc.ssl_target_name_override` | Applied as authority override when `grpc.default_authority` is absent; TLS server-name behavior will be added with TLS credentials. |
| `grpc.primary_user_agent` | Accepted for compatibility; request metadata remains the source of the effective user-agent for now. |
| `grpc.max_receive_message_length` | Enforced with `grpc.MaxCallRecvMsgSize`; `-1` maps to grpc-go's largest supported receive size. |
| `grpc.max_metadata_size` | Applied as grpc-go max header list size when absolute max is absent. |
| `grpc.absolute_max_metadata_size` | Applied as grpc-go max header list size and takes precedence over `grpc.max_metadata_size`. |

The first smoke target is plaintext Spanner emulator. TLS and mTLS credential
mapping are intentionally not production-ready yet.
