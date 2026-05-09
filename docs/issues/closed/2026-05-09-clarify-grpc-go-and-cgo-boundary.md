# Clarify grpc-go And CGo Boundary

## State

closed

## Source

user

## Context

Clarify that this repository should use grpc-go as the gRPC transport implementation and should not depend on gRPC Core. CGo remains relevant for the PHP/Zend native extension boundary and object lifecycle control.

## Acceptance Criteria

- `docs/design.md` distinguishes CGo/Zend bridge responsibilities from grpc-go transport responsibilities.
- `docs/design.md` explains that ext-grpc uses gRPC Core while this design uses grpc-go behind a PHP/Zend bridge.
- FrankenPHP generator usage is described as prototype or boilerplate support, not the assumed lifecycle owner for complex streaming objects.

## Verification

Verified by reading the updated `FrankenPHP Extension Approach` and `Open Design Risks` sections in `docs/design.md`.
