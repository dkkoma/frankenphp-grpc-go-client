# Work: Channel Pool Leases

## Status

Closed on 2026-05-13.

## Context

Creating a new PHP `FrankenGrpc\Channel` for the same target/options previously
created a new grpc-go `ClientConn` every time. `ClientConn` owns resolver,
balancer, HTTP/2 connection, and transport state, so repeated construction could
create unnecessary goroutines, file descriptors, and connection setup work.

## Completed

- Added a process-local `ChannelPool`.
- `fg_channel_new()` now acquires a pooled channel lease instead of dialing a
  fresh physical channel directly.
- Matching target and normalized effective channel options reuse the same
  grpc-go `ClientConn`.
- Different effective options use different `ClientConn` instances.
- `Channel::close()` remains idempotent and now releases only the PHP channel
  lease.
- The physical `ClientConn` closes when the final lease is released.
- Added Go tests for reuse, option separation, and final-lease close behavior.

## Deferred

Idle TTL and LRU eviction are not implemented yet. Refcounted close is enough
for the current FrankenPHP/php-grpc-lite integration path.
