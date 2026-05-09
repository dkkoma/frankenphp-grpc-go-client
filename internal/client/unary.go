package client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type UnaryRequest struct {
	Method         string
	Payload        []byte
	Metadata       Metadata
	TimeoutSeconds *float64
}

type UnaryResult struct {
	Payload          []byte
	Status           Status
	InitialMetadata  Metadata
	TrailingMetadata Metadata
	Peer             string
}

func Unary(ctx context.Context, ch *Channel, req UnaryRequest) (UnaryResult, error) {
	conn, err := ch.Conn()
	if err != nil {
		return UnaryResult{}, err
	}

	ctx, cancel := contextWithTimeout(ctx, req.TimeoutSeconds)
	defer cancel()

	if md := normalizeMetadata(req.Metadata); len(md) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	var response []byte
	var header metadata.MD
	var trailer metadata.MD
	var remotePeer peer.Peer
	err = conn.Invoke(
		ctx,
		req.Method,
		req.Payload,
		&response,
		grpc.ForceCodec(bytesCodec{}),
		grpc.Header(&header),
		grpc.Trailer(&trailer),
		grpc.Peer(&remotePeer),
	)

	if err != nil {
		st, ok := statusFromError(err)
		if !ok {
			return UnaryResult{}, err
		}
		return UnaryResult{
			Status:           st,
			InitialMetadata:  metadataFromGRPC(header),
			TrailingMetadata: metadataFromGRPC(trailer),
			Peer:             peerAddress(remotePeer),
		}, nil
	}

	return UnaryResult{
		Payload:          response,
		Status:           Status{Code: int(codes.OK)},
		InitialMetadata:  metadataFromGRPC(header),
		TrailingMetadata: metadataFromGRPC(trailer),
		Peer:             peerAddress(remotePeer),
	}, nil
}

func peerAddress(remotePeer peer.Peer) string {
	if remotePeer.Addr == nil {
		return ""
	}
	return remotePeer.Addr.String()
}

func contextWithTimeout(ctx context.Context, timeoutSeconds *float64) (context.Context, context.CancelFunc) {
	if timeoutSeconds == nil {
		return context.WithCancel(ctx)
	}

	timeout := time.Duration(*timeoutSeconds * float64(time.Second))
	return context.WithTimeout(ctx, timeout)
}
