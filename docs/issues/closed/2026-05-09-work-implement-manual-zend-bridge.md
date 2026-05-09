# Implement Manual Zend Bridge

## State

closed

## Source

work

## Context

Expose the grpc-go client core through the required `FrankenGrpc` PHP classes
using a manual CGo/Zend bridge.

## Acceptance Criteria

- `FrankenGrpc\Channel`, `UnaryCall`, `ServerStreamingCall`, `UnaryResult`, and
  `Status` are registered by the native extension.
- PHP objects that own native resources store opaque Go handle IDs.
- Go registry owns grpc-go channel, unary call, and server-streaming state.
- Unary calls and server-streaming calls cross the PHP/Zend boundary as raw
  bytes and metadata.
- FrankenPHP binary can be built with this extension.
- PHP smoke verifies class registration and value object behavior.

## Verification

```sh
docker compose run --rm dev make verify
```

Passed.

```sh
docker compose run --rm dev make smoke-api
```

Passed.
