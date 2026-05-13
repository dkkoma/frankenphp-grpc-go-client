package client

import (
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var defaultPool = NewChannelPool()

type ChannelPool struct {
	mu      sync.Mutex
	entries map[channelPoolKey]*channelPoolEntry
}

type channelPoolEntry struct {
	conn        *grpc.ClientConn
	callOptions []grpc.CallOption
	close       func() error
	refs        int
}

type channelPoolKey struct {
	target                    string
	authority                 string
	tlsServerNameOverride     string
	maxReceiveMessageLength   int
	hasMaxReceiveMessage      bool
	maxMetadataSize           int
	hasMaxMetadata            bool
	absoluteMaxMetadataSize   int
	hasAbsoluteMaxMetadata    bool
	hasCredentialsPlaceholder bool
	primaryUserAgent          string
}

func NewChannelPool() *ChannelPool {
	return &ChannelPool{
		entries: map[channelPoolKey]*channelPoolEntry{},
	}
}

func AcquireChannel(target string, config DialConfig) (*Channel, error) {
	return defaultPool.Acquire(target, config)
}

func (p *ChannelPool) Acquire(target string, config DialConfig) (*Channel, error) {
	key := newChannelPoolKey(target, config)

	p.mu.Lock()
	defer p.mu.Unlock()

	if entry := p.entries[key]; entry != nil {
		entry.refs++
		return p.newLease(key, entry), nil
	}

	conn, err := grpc.NewClient(target, dialOptions(config)...)
	if err != nil {
		return nil, err
	}

	entry := &channelPoolEntry{
		conn:        conn,
		callOptions: callOptions(config),
		close:       conn.Close,
		refs:        1,
	}
	p.entries[key] = entry

	return p.newLease(key, entry), nil
}

func (p *ChannelPool) newLease(key channelPoolKey, entry *channelPoolEntry) *Channel {
	return NewChannelWithCallOptions(entry.conn, func() error {
		return p.release(key, entry)
	}, entry.callOptions)
}

func (p *ChannelPool) release(key channelPoolKey, entry *channelPoolEntry) error {
	p.mu.Lock()
	current := p.entries[key]
	if current != entry {
		p.mu.Unlock()
		return nil
	}

	entry.refs--
	if entry.refs > 0 {
		p.mu.Unlock()
		return nil
	}

	delete(p.entries, key)
	close := entry.close
	p.mu.Unlock()

	return close()
}

func dialOptions(config DialConfig, opts ...grpc.DialOption) []grpc.DialOption {
	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if config.Authority != "" {
		dialOptions = append(dialOptions, grpc.WithAuthority(config.Authority))
	} else if config.TLSServerNameOverride != "" {
		dialOptions = append(dialOptions, grpc.WithAuthority(config.TLSServerNameOverride))
	}
	if maxHeaderListSize, ok := maxHeaderListSize(config); ok {
		dialOptions = append(dialOptions, grpc.WithMaxHeaderListSize(uint32(maxHeaderListSize)))
	}
	dialOptions = append(dialOptions, opts...)
	return dialOptions
}

func newChannelPoolKey(target string, config DialConfig) channelPoolKey {
	key := channelPoolKey{
		target:                    target,
		authority:                 config.Authority,
		tlsServerNameOverride:     config.TLSServerNameOverride,
		hasCredentialsPlaceholder: config.HasCredentialsPlaceholder,
		primaryUserAgent:          config.PrimaryUserAgent,
	}
	if config.MaxReceiveMessageLength != nil {
		key.hasMaxReceiveMessage = true
		key.maxReceiveMessageLength = *config.MaxReceiveMessageLength
	}
	if config.MaxMetadataSize != nil {
		key.hasMaxMetadata = true
		key.maxMetadataSize = *config.MaxMetadataSize
	}
	if config.AbsoluteMaxMetadataSize != nil {
		key.hasAbsoluteMaxMetadata = true
		key.absoluteMaxMetadataSize = *config.AbsoluteMaxMetadataSize
	}
	return key
}
