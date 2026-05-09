package client

import (
	"fmt"

	"google.golang.org/grpc/encoding"
)

const codecName = "frankengrpc-bytes"

func init() {
	encoding.RegisterCodec(bytesCodec{})
}

type bytesCodec struct{}

func (bytesCodec) Name() string {
	return codecName
}

func (bytesCodec) Marshal(v any) ([]byte, error) {
	switch value := v.(type) {
	case []byte:
		return cloneBytes(value), nil
	case *[]byte:
		if value == nil {
			return nil, nil
		}
		return cloneBytes(*value), nil
	default:
		return nil, fmt.Errorf("frankengrpc: cannot marshal %T as raw bytes", v)
	}
}

func (bytesCodec) Unmarshal(data []byte, v any) error {
	switch value := v.(type) {
	case *[]byte:
		*value = cloneBytes(data)
		return nil
	default:
		return fmt.Errorf("frankengrpc: cannot unmarshal raw bytes into %T", v)
	}
}

func cloneBytes(data []byte) []byte {
	if data == nil {
		return nil
	}
	cloned := make([]byte, len(data))
	copy(cloned, data)
	return cloned
}
