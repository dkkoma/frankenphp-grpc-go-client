<?php

declare(strict_types=1);

use FrankenGrpc\Channel;
use FrankenGrpc\ServerStreamingCall;
use FrankenGrpc\Status;
use FrankenGrpc\UnaryCall;
use FrankenGrpc\UnaryResult;

assert(class_exists(Channel::class));
assert(class_exists(UnaryCall::class));
assert(class_exists(ServerStreamingCall::class));
assert(class_exists(UnaryResult::class));
assert(class_exists(Status::class));

$status = new Status(0, 'ok', ['x-test' => ['value']]);
assert($status->code === 0);
assert($status->details === 'ok');
assert($status->metadata === ['x-test' => ['value']]);

$result = new UnaryResult('payload', $status, ['h' => ['v']], ['t' => ['v']]);
assert($result->payload === 'payload');
assert($result->status === $status);
assert($result->initialMetadata === ['h' => ['v']]);
assert($result->trailingMetadata === ['t' => ['v']]);

echo "ok\n";
