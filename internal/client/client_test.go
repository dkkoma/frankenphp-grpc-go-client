package client

import (
	"context"
	"io"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func TestUnarySuccessReturnsPayloadAndMetadata(t *testing.T) {
	fixture := newFixture(t, testHandlers{
		unary: func(ctx context.Context, payload []byte) ([]byte, error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				t.Fatal("missing incoming metadata")
			}
			if got := md.Get("x-request"); len(got) != 1 || got[0] != "abc" {
				t.Fatalf("metadata x-request = %v, want [abc]", got)
			}
			if got := md.Get("trace-bin"); len(got) != 1 || got[0] != "\x00\xff" {
				t.Fatalf("metadata trace-bin = %q, want raw binary value", got)
			}

			grpc.SetHeader(ctx, metadata.Pairs("x-header", "h1"))
			grpc.SetTrailer(ctx, metadata.Pairs("x-trailer", "t1"))
			return append([]byte("response:"), payload...), nil
		},
	})

	result, err := Unary(context.Background(), fixture.channel, UnaryRequest{
		Method:   "/frankengrpc.Test/Unary",
		Payload:  []byte("request"),
		Metadata: Metadata{"X-Request": {"abc"}, "trace-bin": {"\x00\xff"}},
	})
	if err != nil {
		t.Fatalf("Unary returned error: %v", err)
	}

	if string(result.Payload) != "response:request" {
		t.Fatalf("payload = %q, want response:request", result.Payload)
	}
	if result.Status.Code != int(codes.OK) {
		t.Fatalf("status code = %d, want OK", result.Status.Code)
	}
	if result.Peer == "" {
		t.Fatal("peer is empty")
	}
	if got := result.InitialMetadata["x-header"]; len(got) != 1 || got[0] != "h1" {
		t.Fatalf("initial metadata = %v, want x-header h1", result.InitialMetadata)
	}
	if got := result.TrailingMetadata["x-trailer"]; len(got) != 1 || got[0] != "t1" {
		t.Fatalf("trailing metadata = %v, want x-trailer t1", result.TrailingMetadata)
	}
}

func TestUnaryNonOKStatusReturnsStatusAndTrailers(t *testing.T) {
	fixture := newFixture(t, testHandlers{
		unary: func(ctx context.Context, payload []byte) ([]byte, error) {
			grpc.SetTrailer(ctx, metadata.Pairs("grpc-status-details-bin", "\x01details"))
			return nil, status.Error(codes.PermissionDenied, "blocked")
		},
	})

	result, err := Unary(context.Background(), fixture.channel, UnaryRequest{
		Method:  "/frankengrpc.Test/Unary",
		Payload: []byte("request"),
	})
	if err != nil {
		t.Fatalf("Unary returned error: %v", err)
	}

	if result.Status.Code != int(codes.PermissionDenied) {
		t.Fatalf("status code = %d, want PermissionDenied", result.Status.Code)
	}
	if result.Status.Details != "blocked" {
		t.Fatalf("status details = %q, want blocked", result.Status.Details)
	}
	if got := result.TrailingMetadata["grpc-status-details-bin"]; len(got) != 1 || got[0] != "\x01details" {
		t.Fatalf("trailing metadata = %v, want status details", result.TrailingMetadata)
	}
}

func TestServerStreamingReadsMessagesAndFinalMetadata(t *testing.T) {
	fixture := newFixture(t, testHandlers{
		stream: func(payload []byte, stream grpc.ServerStream) error {
			grpc.SetHeader(stream.Context(), metadata.Pairs("x-header", "h1"))
			grpc.SetTrailer(stream.Context(), metadata.Pairs("x-trailer", "t1"))
			if err := stream.SendMsg([]byte("first:" + string(payload))); err != nil {
				return err
			}
			return stream.SendMsg([]byte("second"))
		},
	})

	stream, err := StartServerStream(context.Background(), fixture.channel, ServerStreamingRequest{
		Method:  "/frankengrpc.Test/Stream",
		Payload: []byte("request"),
	})
	if err != nil {
		t.Fatalf("StartServerStream returned error: %v", err)
	}

	if got := stream.InitialMetadata()["x-header"]; len(got) != 1 || got[0] != "h1" {
		t.Fatalf("initial metadata = %v, want x-header h1", stream.InitialMetadata())
	}
	message, ok, err := stream.Read()
	if err != nil || !ok || string(message) != "first:request" {
		t.Fatalf("first read = (%q, %v, %v), want first:request true nil", message, ok, err)
	}
	message, ok, err = stream.Read()
	if err != nil || !ok || string(message) != "second" {
		t.Fatalf("second read = (%q, %v, %v), want second true nil", message, ok, err)
	}
	message, ok, err = stream.Read()
	if err != nil || ok || message != nil {
		t.Fatalf("end read = (%q, %v, %v), want nil false nil", message, ok, err)
	}
	if stream.Status().Code != int(codes.OK) {
		t.Fatalf("status code = %d, want OK", stream.Status().Code)
	}
	if got := stream.TrailingMetadata()["x-trailer"]; len(got) != 1 || got[0] != "t1" {
		t.Fatalf("trailing metadata = %v, want x-trailer t1", stream.TrailingMetadata())
	}
}

func TestServerStreamingNonOKAfterMessage(t *testing.T) {
	fixture := newFixture(t, testHandlers{
		stream: func(payload []byte, stream grpc.ServerStream) error {
			if err := stream.SendMsg([]byte("first")); err != nil {
				return err
			}
			grpc.SetTrailer(stream.Context(), metadata.Pairs("grpc-status-details-bin", "\x02details"))
			return status.Error(codes.ResourceExhausted, "quota")
		},
	})

	stream, err := StartServerStream(context.Background(), fixture.channel, ServerStreamingRequest{
		Method:  "/frankengrpc.Test/Stream",
		Payload: []byte("request"),
	})
	if err != nil {
		t.Fatalf("StartServerStream returned error: %v", err)
	}

	message, ok, err := stream.Read()
	if err != nil || !ok || string(message) != "first" {
		t.Fatalf("first read = (%q, %v, %v), want first true nil", message, ok, err)
	}
	_, ok, err = stream.Read()
	if err != nil || ok {
		t.Fatalf("end read = (%v, %v), want false nil", ok, err)
	}
	if stream.Status().Code != int(codes.ResourceExhausted) {
		t.Fatalf("status code = %d, want ResourceExhausted", stream.Status().Code)
	}
	if stream.Status().Details != "quota" {
		t.Fatalf("status details = %q, want quota", stream.Status().Details)
	}
	if got := stream.TrailingMetadata()["grpc-status-details-bin"]; len(got) != 1 || got[0] != "\x02details" {
		t.Fatalf("trailing metadata = %v, want status details", stream.TrailingMetadata())
	}
}

func TestChannelCloseIsIdempotentAndBlocksNewCalls(t *testing.T) {
	fixture := newFixture(t, testHandlers{})

	if err := fixture.channel.Close(); err != nil {
		t.Fatalf("first close returned error: %v", err)
	}
	if err := fixture.channel.Close(); err != nil {
		t.Fatalf("second close returned error: %v", err)
	}

	_, err := Unary(context.Background(), fixture.channel, UnaryRequest{
		Method:  "/frankengrpc.Test/Unary",
		Payload: []byte("request"),
	})
	if err != ErrChannelClosed {
		t.Fatalf("Unary error = %v, want ErrChannelClosed", err)
	}
}

func TestServerStreamingCancelIsIdempotent(t *testing.T) {
	fixture := newFixture(t, testHandlers{
		stream: func(payload []byte, stream grpc.ServerStream) error {
			if err := stream.SendHeader(metadata.Pairs("x-header", "started")); err != nil {
				return err
			}
			<-stream.Context().Done()
			return stream.Context().Err()
		},
	})

	stream, err := StartServerStream(context.Background(), fixture.channel, ServerStreamingRequest{
		Method:  "/frankengrpc.Test/Stream",
		Payload: []byte("request"),
	})
	if err != nil {
		t.Fatalf("StartServerStream returned error: %v", err)
	}

	stream.Cancel()
	stream.Cancel()

	if stream.Status().Code != int(codes.Canceled) {
		t.Fatalf("status code = %d, want Canceled", stream.Status().Code)
	}
	message, ok, err := stream.Read()
	if err != nil || ok || message != nil {
		t.Fatalf("read after cancel = (%q, %v, %v), want nil false nil", message, ok, err)
	}
}

func TestUnaryDeadlineExceeded(t *testing.T) {
	fixture := newFixture(t, testHandlers{
		unary: func(ctx context.Context, payload []byte) ([]byte, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})
	timeout := 0.01

	result, err := Unary(context.Background(), fixture.channel, UnaryRequest{
		Method:         "/frankengrpc.Test/Unary",
		Payload:        []byte("request"),
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatalf("Unary returned error: %v", err)
	}
	if result.Status.Code != int(codes.DeadlineExceeded) {
		t.Fatalf("status code = %d, want DeadlineExceeded", result.Status.Code)
	}
}

type testFixture struct {
	channel *Channel
	server  *grpc.Server
}

type testHandlers struct {
	unary  func(context.Context, []byte) ([]byte, error)
	stream func([]byte, grpc.ServerStream) error
}

func newFixture(t *testing.T, handlers testHandlers) testFixture {
	t.Helper()

	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer(grpc.ForceServerCodec(bytesCodec{}))
	RegisterTestService(server, handlers)
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Logf("test server stopped: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	channel := NewChannel(conn, func() error {
		err := conn.Close()
		server.Stop()
		return err
	})
	t.Cleanup(func() {
		channel.Close()
		listener.Close()
	})

	return testFixture{channel: channel, server: server}
}

func RegisterTestService(server *grpc.Server, handlers testHandlers) {
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "frankengrpc.Test",
		HandlerType: (*testServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Unary",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					var payload []byte
					if err := dec(&payload); err != nil {
						return nil, err
					}
					handler := handlers.unary
					if handler == nil {
						handler = func(context.Context, []byte) ([]byte, error) {
							return nil, status.Error(codes.Unimplemented, "unimplemented")
						}
					}
					if interceptor == nil {
						return handler(ctx, payload)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/frankengrpc.Test/Unary",
					}
					return interceptor(ctx, payload, info, func(ctx context.Context, req any) (any, error) {
						bytes, ok := req.([]byte)
						if !ok {
							return nil, status.Error(codes.Internal, "unexpected request type")
						}
						return handler(ctx, bytes)
					})
				},
			},
		},
		Streams: []grpc.StreamDesc{
			{
				StreamName:    "Stream",
				ServerStreams: true,
				Handler: func(srv any, stream grpc.ServerStream) error {
					var payload []byte
					if err := stream.RecvMsg(&payload); err != nil {
						if err == io.EOF {
							return status.Error(codes.InvalidArgument, "missing request payload")
						}
						return err
					}
					handler := handlers.stream
					if handler == nil {
						handler = func([]byte, grpc.ServerStream) error {
							return status.Error(codes.Unimplemented, "unimplemented")
						}
					}
					return handler(payload, stream)
				},
			},
		},
	}, testService{})
}

type testService struct{}

type testServiceServer interface {
	testService()
}

func (testService) testService() {}
