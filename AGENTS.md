# Repository Guidelines

## Project Structure & Module Organization

This repository hosts a FrankenPHP PHP extension implemented in Go, backed by `grpc-go`. Its primary consumer is `~/src/php-grpc-lite-gax`, which expects the PHP surface documented in that repository's `docs/frankenphp-extension-api.md`.

Keep Go implementation code under `internal/` and public package entry points at the module root or `build/` only when required by FrankenPHP/Caddy module integration. Keep PHP stubs, examples, and compatibility probes under `stubs/`, `examples/`, and `tests/php/`. Keep design documents under `docs/`. Use `docs/issues/open/` for active findings and move resolved items to `docs/issues/closed/`. Prefer one Markdown file per issue, named like `metadata-binary-roundtrip.md`.

## Build, Test, and Development Commands

All build, test, and smoke commands should run inside the dev container. Prefer Docker Compose entry points from the host and `make` targets inside the container.

- `docker compose run --rm dev make test`: run Go unit tests.
- `docker compose run --rm dev make test-race`: run Go tests with race detection for channel and stream lifecycle work.
- `docker compose run --rm dev make lint-stubs`: lint PHP API stubs.
- `docker compose run --rm dev make test-extension`: compile-check the FrankenPHP extension bridge.
- `docker compose run --rm dev make verify`: run the standard verification suite.
- `docker compose run --rm dev make build-frankenphp`: build `bin/frankenphp` with this extension.
- `docker compose run --rm dev go test -tags=integration ./...`: run integration tests that require a built FrankenPHP binary or emulator.
- `docker compose run --rm dev php -d extension=...`: run PHP-level smoke tests when a loadable test binary exists.
- `docker compose run --rm dev xcaddy build --with <module-or-build-path>`: build a FrankenPHP binary containing this extension once `xcaddy` is installed in the container image.

When commands are added to a `Makefile`, `justfile`, or scripts directory, document and prefer those commands here.

## Coding Style & Naming Conventions

Use idiomatic Go with small packages and explicit ownership boundaries. Keep grpc-go client behavior separate from Zend/PHP value conversion and from FrankenPHP module registration. Avoid exposing grpc-go or generated protobuf types through the PHP boundary; all request and response messages crossing the PHP API are serialized protobuf bytes.

The PHP namespace exposed by the extension is `FrankenGrpc`. Preserve the consumer-facing class names and method signatures from `php-grpc-lite-gax/docs/frankenphp-extension-api.md`: `Channel`, `UnaryCall`, `ServerStreamingCall`, `UnaryResult`, and `Status`.

Use lowercase gRPC metadata keys internally. Preserve binary metadata values exactly for `-bin` keys. Do not manufacture successful statuses when native state is malformed; return a PHP exception instead.

## Testing Guidelines

Test behavior at both boundaries:

- Go unit tests for grpc-go request construction, metadata normalization, deadline/cancel handling, status conversion, and channel lifecycle.
- PHP smoke tests for the exact exposed `FrankenGrpc` classes and method signatures.
- Emulator integration tests for unary and server streaming behavior against real gRPC services when feasible.

Cover unary success, unary non-OK status, server streaming with multiple messages, non-OK final stream status after messages, `getInitialMetadata()` before `read()`, deadline exceeded, cancellation, channel close idempotency, and binary metadata preservation.

## Design Docs & Issue Tracking

Keep architecture notes, API sketches, and tradeoff records in `docs/design.md`. Update it to the latest current design only; avoid stale historical alternatives. Every user instruction, work unit, and reviewer finding should have its own Markdown issue file. Use `docs/issues/template.md` for new issues. Create active items under `docs/issues/open/`; after implementation and verification, move the file to `docs/issues/closed/` and update its state.

## Review Workflow

After implementation, run focused reviews before closing work. At minimum, include a domain-model review for naming, responsibilities, invariants, lifecycle, status semantics, and public/internal boundaries. Add reviewers for concurrency, CGo/Zend memory ownership, tests, and maintainability when those areas are touched.

## Commit & Pull Request Guidelines

Use concise imperative commits, optionally with Conventional Commit prefixes such as `feat:`, `fix:`, `test:`, and `docs:`. Pull requests should summarize behavior changes, list verification commands, link related `docs/issues` files, and call out public API, binary compatibility, or FrankenPHP build impact.
