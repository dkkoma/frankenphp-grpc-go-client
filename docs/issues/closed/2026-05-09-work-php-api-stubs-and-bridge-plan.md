# PHP API Stubs And Bridge Plan

## State

closed

## Source

work

## Context

Add repository artifacts that make the intended `FrankenGrpc` PHP API and the
native bridge boundary explicit before implementing the CGo/Zend bridge.

## Acceptance Criteria

- PHP stubs define `FrankenGrpc\Channel`, `UnaryCall`, `ServerStreamingCall`,
  `UnaryResult`, and `Status`.
- Bridge plan maps PHP methods to `internal/client` types.
- Spanner emulator is recorded as the first repository-level smoke target.

## Verification

```sh
docker compose run --rm dev php -l stubs/FrankenGrpc.php
```

Passed.
