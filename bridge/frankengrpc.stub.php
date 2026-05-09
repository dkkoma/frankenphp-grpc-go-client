<?php

/** @generate-class-entries */

namespace FrankenGrpc;

final class Channel
{
    public function __construct(string $target, array $options = []) {}
    public function close(): void {}
}

final class UnaryCall
{
    public function __construct(Channel $channel, string $method) {}

    public function start(
        string $payload,
        array $metadata = [],
        ?float $timeoutSeconds = null,
    ): UnaryResult {}

    public function cancel(): void {}
    public function getPeer(): string {}
}

final class ServerStreamingCall
{
    public function __construct(Channel $channel, string $method) {}

    public function start(
        string $payload,
        array $metadata = [],
        ?float $timeoutSeconds = null,
    ): void {}

    public function read(): ?string {}
    public function getInitialMetadata(): array {}
    public function getStatus(): Status {}
    public function getTrailingMetadata(): array {}
    public function cancel(): void {}
    public function getPeer(): string {}
}

final class UnaryResult
{
    public string $payload;
    public Status $status;
    public array $initialMetadata;
    public array $trailingMetadata;

    public function __construct(
        string $payload,
        Status $status,
        array $initialMetadata = [],
        array $trailingMetadata = [],
    ) {}
}

final class Status
{
    public int $code;
    public string $details;
    public array $metadata;

    public function __construct(
        int $code,
        string $details = '',
        array $metadata = [],
    ) {}
}
