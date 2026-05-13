# Design

## Goal

This repository provides a FrankenPHP-integrated PHP extension implemented in Go. The extension exposes byte-oriented gRPC primitives to PHP and uses `grpc-go` internally. The primary consumer is `~/src/php-grpc-lite-gax`, which adapts this extension to `google/gax` transports.

The extension must expose the PHP API described by `php-grpc-lite-gax/docs/frankenphp-extension-api.md` and must not depend on `google/gax`, google-cloud-php clients, Composer packages, or generated protobuf classes.

## Public PHP Surface

The PHP namespace is `FrankenGrpc`.

Required classes:

- `Channel`
- `UnaryCall`
- `ServerStreamingCall`
- `UnaryResult`
- `Status`

Request and response payloads are serialized protobuf byte strings. Metadata is represented as `array<string, list<string>>`. Status codes use canonical gRPC integer codes.

The extension treats method paths as opaque full gRPC paths, for example:

```text
/google.pubsub.v1.Publisher/Publish
```

Validation at the extension boundary should focus on type shape, lifecycle, and impossible native state. Higher-level policy validation, such as rejecting reserved request metadata, remains the PHP adapter's responsibility in `php-grpc-lite-gax`.

## FrankenPHP Extension Approach

FrankenPHP currently documents two Go extension paths: a generator-based path and a manual CGo/Zend bridge path. The transport implementation must use `grpc-go`, not gRPC Core. CGo is needed for the PHP/Zend native extension boundary, not for gRPC transport.

This extension uses a manual CGo/Zend bridge because the public surface returns and accepts extension objects (`Channel`, `UnaryCall`, `ServerStreamingCall`, `UnaryResult`, `Status`) and must manage long-lived native resources. The generator remains useful as reference material, but it does not currently cover object parameters and object return values needed by this API.

The design therefore assumes:

- Go owns the grpc-go client connection, call state, stream state, contexts, and cancellation functions.
- PHP objects are opaque handles over Go-owned resources.
- Zend/CGo glue owns PHP object allocation, argument parsing, return value construction, and exception throwing.
- The Go packages remain testable without loading PHP.

The key design choice is not "CGo versus grpc-go". The intended stack is:

```text
PHP / Zend Engine
  -> CGo / Zend bridge
  -> Go lifecycle and handle registry
  -> grpc-go
  -> remote gRPC server
```

This is different from ext-grpc:

```text
PHP
  -> ext-grpc
  -> gRPC Core
  -> remote gRPC server
```

## Internal Package Layout

Planned layout:

```text
.
+-- build/                  # FrankenPHP/Caddy module integration if generated or required
+-- docs/
|   +-- design.md
|   +-- issues/
+-- internal/
|   +-- client/             # grpc-go channel, dial options, unary and stream execution
|   +-- phptypes/           # PHP metadata/status/result conversion helpers
|   +-- registry/           # handle registry and lifecycle guards for PHP objects
|   +-- testservice/        # in-process gRPC test service for Go tests
+-- stubs/                  # PHP stubs for IDEs and API compatibility checks
+-- tests/
|   +-- go/
|   +-- php/
+-- go.mod
```

The root package contains the FrankenPHP extension registration behind the `frankengrpc_extension` build tag. Normal Go unit tests run without importing FrankenPHP. FrankenPHP binary builds use `xcaddy` with the extension tag enabled.

## Runtime Model

`Channel` maps to a lease on a pooled grpc-go `ClientConn`. The pool key is the target plus normalized effective channel options, so repeated PHP channel construction for the same endpoint reuses resolver, HTTP/2 connection, and balancer state. Unknown PHP channel options are ignored for compatibility with `Grpc\Channel`. The implemented option subset accepts `credentials` as a placeholder, applies authority overrides, enforces max receive message size, and maps metadata-size limits to grpc-go max header list size. Plaintext transport is the only supported transport for the first Spanner emulator smoke; TLS and mTLS credential mapping remain explicit production gaps. `close()` is idempotent and releases the PHP object's lease; the physical `ClientConn` is closed only when the final lease is released. Calls started after close throw a PHP exception. Calls already in flight rely on grpc-go behavior and must not crash PHP.

`UnaryCall` references a `Channel` and a method path. `start()` creates a context, applies outgoing metadata and optional deadline, invokes grpc-go with raw request and response byte buffers, then returns `UnaryResult`.

`ServerStreamingCall` references a `Channel` and a method path. `start()` creates a context, sends one raw request message, half-closes the client side, captures initial metadata, and keeps stream state. `read()` returns the next serialized response payload or `null` at end of stream. Final status and trailers are cached after stream completion.

`cancel()` cancels the active context. If cancellation wins before grpc-go has a final status, the observable status should be `CANCELLED` when grpc-go exposes it.

## grpc-go Mapping

The implementation uses raw protobuf bytes rather than generated message types. The client package should provide a codec or marshaler path suitable for `[]byte` payloads, so PHP stays responsible for protobuf serialization and decoding.

Metadata conversion rules:

- Normalize metadata keys to lowercase before passing to grpc-go.
- Preserve value order.
- Preserve binary values exactly for keys ending in `-bin`.
- Return initial and trailing metadata as `array<string, list<string>>`.
- Include status detail trailers such as `grpc-status-details-bin` when grpc-go exposes them.

Deadline conversion:

- `null` timeout means no explicit deadline.
- Non-null timeout is relative to call start.
- Deadline expiry maps to canonical `DEADLINE_EXCEEDED`.

Error conversion:

- gRPC application statuses are returned as `Status`.
- Transport failures before any gRPC status is available throw PHP exceptions.
- Malformed native state throws PHP exceptions.

## Concurrency And Memory Ownership

Native handles must be safe against repeated close, cancellation races, stream reads after completion, and PHP object destruction. A handle registry should make ownership explicit and prevent use-after-free across PHP-visible objects.

Large payloads should cross the PHP/Go boundary with a clear copy policy. The initial implementation should prefer correctness and explicit copies over borrowed memory unless profiling proves the copy cost matters.

No goroutine should outlive its channel or call without a reachable cancellation path. Race tests should cover close/cancel/read interactions.

## Test Strategy

The first implementation milestone should be test-first around `internal/client` using an in-process grpc-go service:

- Unary success with headers and trailers.
- Unary non-OK status with details and trailers.
- Server streaming success with multiple messages.
- Server streaming non-OK final status after at least one message.
- Deadline exceeded.
- Cancellation.
- Binary metadata round trip.

PHP-level tests should then verify the exact `FrankenGrpc` surface:

- Constructor and method signatures.
- `getInitialMetadata()` before `read()` does not consume the first response.
- Repeated metadata/status/trailer access is idempotent.
- `Channel::close()` is idempotent.
- Calls after close fail predictably.

Finally, `php-grpc-lite-gax` should add or reuse FrankenPHP bridge smoke tests equivalent to its current native, Pub/Sub emulator, and Spanner emulator smoke paths.

## Milestones

1. Scaffold Go module, PHP stubs, and a minimal FrankenPHP build integration.
2. Implement raw-byte grpc-go unary client with status and metadata mapping.
3. Implement `Channel`, `Status`, and `UnaryResult` PHP objects.
4. Implement `UnaryCall::start()`, `cancel()`, and `getPeer()`.
5. Implement raw-byte server streaming client state machine.
6. Implement `ServerStreamingCall` PHP object and lifecycle semantics.
7. Add PHP smoke tests against a built FrankenPHP binary.
8. Integrate smoke verification from `php-grpc-lite-gax`.

## Open Design Risks

- FrankenPHP generator support may not be sufficient for this object-heavy API; manual CGo/Zend glue is the assumed path for lifecycle-sensitive objects until a prototype proves otherwise.
- grpc-go raw `[]byte` codec behavior must be validated early to avoid accidentally depending on generated protobuf messages.
- PHP arrays containing binary strings require careful conversion to avoid invalid UTF-8 assumptions.
- Resource lifetime across PHP request shutdown, channel close, stream cancellation, and goroutine cleanup needs explicit tests.
- Build reproducibility depends on matching PHP headers, FrankenPHP, CGO flags, and the target platform.

## References

- `~/src/php-grpc-lite-gax/docs/frankenphp-extension-api.md`
- `~/src/php-grpc-lite-gax/docs/design.md`
- FrankenPHP documentation: https://frankenphp.dev/docs/extensions/

## Current Scope

The current implementation includes a dev-container-first workflow, raw-byte grpc-go unary and server-streaming client core, Go handle registry, manual CGo/Zend bridge, `FrankenGrpc` PHP classes, FrankenPHP binary build target, PHP API smoke test, and verification targets. Spanner emulator smoke remains the first external integration target.
