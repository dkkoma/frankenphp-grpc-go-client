# Domain Model Review: Zend Bridge

## State

closed

## Source

review

## Reviewer Role

domain-model-review

## Findings

### Blocker

none

### High

none

### Medium

none

### Low

none

### Design Decision

- Problem: native `Status` and `UnaryResult` properties are declared as public
  untyped Zend properties while PHP stubs use typed readonly properties.
- Evidence: `frankengrpc.c` property declarations and `stubs/FrankenGrpc.php`.
- Domain concept: PHP public API compatibility.
- Why it matters: typed readonly properties would more closely mirror the PHP
  spec, but PHP internal property initialization across manual C code is more
  sensitive and the bridge currently prioritizes stable native object creation.
- Decision: accept for the current bridge milestone because constructor
  signatures, property names, value shapes, and return objects match the
  consumer contract. Revisit before publishing a stable binary API.

## Review Notes

During review, `UnaryCall::cancel()` was found to be a no-op. That violated the
public lifecycle model, so the bridge now stores an in-flight unary context
cancel function and calls it from `UnaryCall::cancel()`.

## Verification

```sh
docker compose run --rm dev make verify
```

Passed.

```sh
docker compose run --rm dev make smoke-api
```

Passed.
