# Spanner Emulator Smoke Through Consumer

## State

open

## Source

work

## Context

This repository now builds a FrankenPHP binary exposing the `FrankenGrpc` PHP
classes. The first external smoke target should be Spanner emulator, but the
consumer repository currently has its Spanner smoke wired to
`GrpcLiteTransport::build()` and its FrankenPHP backend path is unary-only.

## Acceptance Criteria

- `~/src/php-grpc-lite-gax` provides a FrankenPHP transport path that can use
  this extension for unary and server-streaming calls.
- The Spanner emulator smoke can run through the FrankenPHP grpc-go extension,
  covering DML and server-streaming `ExecuteStreamingSql`.
- This repository documents and/or scripts the command used to run that smoke
  from the dev container.

## Verification

Not run yet.
