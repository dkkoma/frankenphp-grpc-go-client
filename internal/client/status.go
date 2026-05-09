package client

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Status struct {
	Code    int
	Details string
}

func statusFromError(err error) (Status, bool) {
	if errors.Is(err, context.DeadlineExceeded) {
		return Status{Code: int(codes.DeadlineExceeded), Details: context.DeadlineExceeded.Error()}, true
	}
	if errors.Is(err, context.Canceled) {
		return Status{Code: int(codes.Canceled), Details: context.Canceled.Error()}, true
	}

	st, ok := status.FromError(err)
	if !ok {
		return Status{}, false
	}
	return Status{
		Code:    int(st.Code()),
		Details: st.Message(),
	}, true
}
