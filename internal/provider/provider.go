package provider

import "context"

type ChunkStream interface {
	Next() (Chunk, error)
	Close() error
}

type Provider interface {
	Name() string
	Stream(ctx context.Context, req Request) (ChunkStream, error)
}
