package client

import (
	"context"
	"errors"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type ServerStreamingRequest struct {
	Method         string
	Payload        []byte
	Metadata       Metadata
	TimeoutSeconds *float64
}

type ServerStream struct {
	stream grpc.ClientStream
	cancel context.CancelFunc

	mu               sync.Mutex
	initialMetadata  Metadata
	trailingMetadata Metadata
	status           *Status
	peer             string
	done             bool
}

func StartServerStream(ctx context.Context, ch *Channel, req ServerStreamingRequest) (*ServerStream, error) {
	conn, err := ch.Conn()
	if err != nil {
		return nil, err
	}
	callOptions, err := ch.CallOptions()
	if err != nil {
		return nil, err
	}

	ctx, cancel := contextWithTimeout(ctx, req.TimeoutSeconds)
	if md := normalizeMetadata(req.Metadata); len(md) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	desc := &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: false,
	}
	var remotePeer peer.Peer
	callOptions = append(callOptions, grpc.ForceCodec(bytesCodec{}), grpc.Peer(&remotePeer))
	stream, err := conn.NewStream(ctx, desc, req.Method, callOptions...)
	if err != nil {
		cancel()
		return nil, err
	}

	if err := stream.SendMsg(req.Payload); err != nil {
		cancel()
		return nil, err
	}
	if err := stream.CloseSend(); err != nil {
		cancel()
		return nil, err
	}

	header, err := stream.Header()
	if err != nil {
		cancel()
		st, ok := statusFromError(err)
		if !ok {
			return nil, err
		}
		return &ServerStream{
			stream:           stream,
			cancel:           cancel,
			initialMetadata:  nil,
			trailingMetadata: metadataFromGRPC(stream.Trailer()),
			status:           &st,
			peer:             peerAddress(remotePeer),
			done:             true,
		}, nil
	}

	return &ServerStream{
		stream:          stream,
		cancel:          cancel,
		initialMetadata: metadataFromGRPC(header),
		peer:            peerAddress(remotePeer),
	}, nil
}

func (s *ServerStream) Read() ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return nil, false, nil
	}

	var response []byte
	err := s.stream.RecvMsg(&response)
	if err == nil {
		return response, true, nil
	}

	s.done = true
	s.trailingMetadata = metadataFromGRPC(s.stream.Trailer())
	var st Status
	if errors.Is(err, io.EOF) {
		st = Status{Code: int(codes.OK)}
	} else {
		statusValue, ok := statusFromError(err)
		if !ok {
			s.cancel()
			return nil, false, err
		}
		st = statusValue
	}
	s.status = &st
	s.cancel()

	return nil, false, nil
}

func (s *ServerStream) InitialMetadata() Metadata {
	s.mu.Lock()
	defer s.mu.Unlock()

	return cloneMetadata(s.initialMetadata)
}

func (s *ServerStream) TrailingMetadata() Metadata {
	s.mu.Lock()
	defer s.mu.Unlock()

	return cloneMetadata(s.trailingMetadata)
}

func (s *ServerStream) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == nil {
		return Status{Code: int(codes.OK)}
	}

	return *s.status
}

func (s *ServerStream) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return
	}

	s.done = true
	st := Status{Code: int(codes.Canceled), Details: context.Canceled.Error()}
	s.status = &st
	s.cancel()
}

func (s *ServerStream) Peer() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.peer
}

func cloneMetadata(in Metadata) Metadata {
	if len(in) == 0 {
		return nil
	}

	out := Metadata{}
	for key, values := range in {
		copied := make([]string, len(values))
		copy(copied, values)
		out[key] = copied
	}
	return out
}
