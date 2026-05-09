# FrankenPHP Bridge Plan

## Purpose

This document records the native bridge work needed to expose `internal/client`
through the `FrankenGrpc` PHP API.

## Boundary

The Go gRPC implementation is already isolated in `internal/client`. The bridge
must translate between PHP/Zend values and these Go types:

- `client.Channel`
- `client.UnaryRequest`
- `client.UnaryResult`
- `client.ServerStreamingRequest`
- `client.ServerStream`
- `client.Metadata`
- `client.Status`

## Object Handles

PHP objects should be opaque handles over Go-owned resources. The bridge should
own a registry with typed handles for:

- channel handles
- unary call handles
- server streaming call handles

Each handle needs a mutex-protected lifecycle state. PHP object destruction must
release the handle and call the relevant close/cancel path exactly once.

## PHP Methods

`Channel::__construct(string $target, array $options = [])` should create a
`client.Channel`. Initial implementation should support plaintext endpoints for
emulator smoke tests. TLS and advanced dial options can be added after the
Spanner smoke path is stable.

`Channel::close()` should call `client.Channel.Close()` and be idempotent.

`UnaryCall::__construct(Channel $channel, string $method)` should retain a
channel handle and method path.

`UnaryCall::start()` should convert PHP bytes and metadata to `client.Unary`,
then return a `UnaryResult` object. gRPC status responses must become `Status`.
Transport failures before status must throw a PHP exception.

`ServerStreamingCall::start()` should create and retain `client.ServerStream`.
`read()` should call `ServerStream.Read()` and return `string|null`.
Metadata and status accessors should return cached values from the stream.

## First Smoke Target

The first repository-level smoke target is the Spanner emulator. It should use
the PHP consumer repository's FrankenPHP backend path and run from the dev
container. It should verify at least:

- unary admin or session call through `UnaryCall`
- server-streaming SQL call through `ServerStreamingCall`
- initial metadata, final status, and trailers do not break the GAX adapter

Pub/Sub emulator smoke can be added after Spanner is green.
