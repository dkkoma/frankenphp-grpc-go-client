# Work: Lightweight UnaryCall Objects

## Status

Closed on 2026-05-14.

## Context

`FrankenGrpc\UnaryCall` previously created Go-side call state in its
constructor and stored that state in the global registry. That made call object
construction cross the CGo boundary even before a request was started.

## Completed

- `UnaryCall::__construct()` now stores the PHP `Channel` object and method
  string in Zend object state only.
- `UnaryCall::start()` calls into Go directly with the channel handle and method
  string.
- Removed Go-side unary call registry handles and constructor/destructor CGo
  calls.
- `UnaryCall::getPeer()` now returns the last peer stored on the PHP object
  after `start()`.
- `UnaryCall::cancel()` remains a no-op for compatibility in this lightweight
  unary model.
- PHP API smoke covers lightweight construction, empty peer, and cancel.

## Notes

This preserves the public `UnaryCall` API while reducing one CGo crossing and
one registry entry per unary call object. A future `Channel::unary()` fast path
can still remove the PHP `UnaryCall` object entirely if benchmarks show that is
worthwhile.
