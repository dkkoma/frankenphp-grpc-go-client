<?php

declare(strict_types=1);

namespace FrankenGrpc;

final class Channel
{
    /**
     * @param array<string, mixed> $options
     */
    public function __construct(string $target, array $options = [])
    {
    }

    public function close(): void
    {
    }
}

final class UnaryCall
{
    public function __construct(Channel $channel, string $method)
    {
    }

    /**
     * @param array<string, list<string>> $metadata
     */
    public function start(
        string $payload,
        array $metadata = [],
        ?float $timeoutSeconds = null,
    ): UnaryResult {
    }

    public function cancel(): void
    {
    }

    public function getPeer(): string
    {
    }
}

final class ServerStreamingCall
{
    public function __construct(Channel $channel, string $method)
    {
    }

    /**
     * @param array<string, list<string>> $metadata
     */
    public function start(
        string $payload,
        array $metadata = [],
        ?float $timeoutSeconds = null,
    ): void {
    }

    public function read(): ?string
    {
    }

    /**
     * @return array<string, list<string>>
     */
    public function getInitialMetadata(): array
    {
    }

    public function getStatus(): Status
    {
    }

    /**
     * @return array<string, list<string>>
     */
    public function getTrailingMetadata(): array
    {
    }

    public function cancel(): void
    {
    }

    public function getPeer(): string
    {
    }
}

final readonly class UnaryResult
{
    /**
     * @param array<string, list<string>> $initialMetadata
     * @param array<string, list<string>> $trailingMetadata
     */
    public function __construct(
        public string $payload,
        public Status $status,
        public array $initialMetadata = [],
        public array $trailingMetadata = [],
    ) {
    }
}

final readonly class Status
{
    /**
     * @param array<string, list<string>> $metadata
     */
    public function __construct(
        public int $code,
        public string $details = '',
        public array $metadata = [],
    ) {
    }
}
