# Work: php-grpc-lite Channel Options

## Status

Closed on 2026-05-09.

## Context

`php-grpc-lite` forwards the original `Grpc\Channel` option array when using the
FrankenPHP grpc-go backend. The extension must tolerate options produced for
`ext-grpc` and map the subset needed by the first consumer smoke path.

## Completed

- Unknown channel options are ignored.
- `credentials` is accepted as a compatibility placeholder without crashing.
- `grpc.default_authority` and `grpc.ssl_target_name_override` are converted to
  grpc-go authority overrides.
- `grpc.primary_user_agent` is accepted as a compatibility placeholder.
- `grpc.max_receive_message_length` is enforced with grpc-go per-call receive
  limits, including `-1` as the largest supported grpc-go receive size.
- `grpc.max_metadata_size` and `grpc.absolute_max_metadata_size` are converted
  to grpc-go max header list size.
- PHP API smoke now creates a channel with representative options.
- Go client tests cover the receive-size limit behavior.

## Remaining Production Gap

TLS and mTLS credential mapping are not implemented. The current implementation
is scoped to plaintext Spanner emulator smoke.
