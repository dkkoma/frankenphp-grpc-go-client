package client

import (
	"errors"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ErrChannelClosed = errors.New("frankengrpc: channel is closed")

type Channel struct {
	conn  *grpc.ClientConn
	mu    sync.RWMutex
	close func() error
	done  bool
}

func Dial(target string, opts ...grpc.DialOption) (*Channel, error) {
	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	dialOptions = append(dialOptions, opts...)

	conn, err := grpc.NewClient(target, dialOptions...)
	if err != nil {
		return nil, err
	}

	return NewChannel(conn, conn.Close), nil
}

func NewChannel(conn *grpc.ClientConn, close func() error) *Channel {
	if close == nil {
		close = func() error { return nil }
	}
	return &Channel{
		conn:  conn,
		close: close,
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

func (c *Channel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.done {
		return nil
	}

	c.done = true
	return c.close()
}
