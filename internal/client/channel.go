package client

import (
	"errors"
	"math"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ErrChannelClosed = errors.New("frankengrpc: channel is closed")

type Channel struct {
	conn        *grpc.ClientConn
	mu          sync.RWMutex
	close       func() error
	callOptions []grpc.CallOption
	done        bool
}

type DialConfig struct {
	Authority                 string
	TLSServerNameOverride     string
	MaxReceiveMessageLength   *int
	MaxMetadataSize           *int
	AbsoluteMaxMetadataSize   *int
	HasCredentialsPlaceholder bool
	PrimaryUserAgent          string
}

func Dial(target string, opts ...grpc.DialOption) (*Channel, error) {
	return DialWithConfig(target, DialConfig{}, opts...)
}

func DialWithConfig(target string, config DialConfig, opts ...grpc.DialOption) (*Channel, error) {
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

	conn, err := grpc.NewClient(target, dialOptions...)
	if err != nil {
		return nil, err
	}

	return NewChannelWithCallOptions(conn, conn.Close, callOptions(config)), nil
}

func NewChannel(conn *grpc.ClientConn, close func() error) *Channel {
	return NewChannelWithCallOptions(conn, close, nil)
}

func NewChannelWithCallOptions(conn *grpc.ClientConn, close func() error, callOptions []grpc.CallOption) *Channel {
	if close == nil {
		close = func() error { return nil }
	}
	return &Channel{
		conn:        conn,
		close:       close,
		callOptions: cloneCallOptions(callOptions),
	}
}

func (c *Channel) Conn() (*grpc.ClientConn, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.done {
		return nil, ErrChannelClosed
	}

	return c.conn, nil
}

func (c *Channel) CallOptions() ([]grpc.CallOption, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.done {
		return nil, ErrChannelClosed
	}

	return cloneCallOptions(c.callOptions), nil
}

func (c *Channel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.done {
		return nil
	}

	c.done = true
	return c.close()
}

func callOptions(config DialConfig) []grpc.CallOption {
	if config.MaxReceiveMessageLength == nil {
		return nil
	}
	if *config.MaxReceiveMessageLength < 0 {
		return []grpc.CallOption{grpc.MaxCallRecvMsgSize(math.MaxInt32)}
	}
	return []grpc.CallOption{grpc.MaxCallRecvMsgSize(*config.MaxReceiveMessageLength)}
}

func maxHeaderListSize(config DialConfig) (int, bool) {
	switch {
	case config.AbsoluteMaxMetadataSize != nil:
		return *config.AbsoluteMaxMetadataSize, true
	case config.MaxMetadataSize != nil:
		return *config.MaxMetadataSize, true
	default:
		return 0, false
	}
}

func cloneCallOptions(options []grpc.CallOption) []grpc.CallOption {
	if len(options) == 0 {
		return nil
	}
	cloned := make([]grpc.CallOption, len(options))
	copy(cloned, options)
	return cloned
}
