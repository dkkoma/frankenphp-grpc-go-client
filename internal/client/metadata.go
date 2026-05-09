package client

import (
	"strings"

	"google.golang.org/grpc/metadata"
)

type Metadata map[string][]string

func normalizeMetadata(in Metadata) metadata.MD {
	if len(in) == 0 {
		return nil
	}

	out := metadata.MD{}
	for key, values := range in {
		normalizedKey := strings.ToLower(key)
		copied := make([]string, len(values))
		copy(copied, values)
		out[normalizedKey] = append(out[normalizedKey], copied...)
	}

	return out
}

func metadataFromGRPC(in metadata.MD) Metadata {
	if len(in) == 0 {
		return nil
	}

	out := Metadata{}
	for key, values := range in {
		copied := make([]string, len(values))
		copy(copied, values)
		out[strings.ToLower(key)] = copied
	}

	return out
}
