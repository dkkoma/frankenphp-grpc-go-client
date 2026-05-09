# Dev Container First Workflow

## State

closed

## Source

user

## Context

Switch repository development, build, and verification workflows to assume the
dev container. Host-local Go and PHP commands should not be the documented
default.

## Acceptance Criteria

- Dev container and Docker Compose configuration exist.
- `AGENTS.md` documents Docker Compose commands as the standard entry points.
- `README.md` documents container-based test and lint commands.
- Existing verification records use container commands.

## Verification

```sh
docker compose run --rm dev make verify
```

Passed.

```sh
docker compose run --rm dev make smoke-api
```

Passed.
