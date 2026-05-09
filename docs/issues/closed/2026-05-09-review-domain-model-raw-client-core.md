# Domain Model Review: Raw Client Core

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

none

## Review Notes

The review found one error-taxonomy issue during the pass: non-gRPC errors were
initially collapsed into `UNKNOWN` status values. That violated the documented
boundary where protocol-level gRPC statuses are values and failures without a
gRPC status remain errors for the PHP bridge to map to exceptions. The issue was
fixed before this review was closed.

## Verification

```sh
docker compose run --rm dev go test ./...
```

Passed.

```sh
docker compose run --rm dev go test -race ./...
```

Passed.
