package contracts

import (
	"context"
	"errors"
	"testing"

	"github.com/open-mrp/api/shared/appctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// fakeServerTransportStream captures the header metadata a handler sets, standing
// in for the real server stream that grpc.SetHeader writes to.
type fakeServerTransportStream struct {
	header  metadata.MD
	trailer metadata.MD
}

func (f *fakeServerTransportStream) Method() string { return "/svc/Method" }

func (f *fakeServerTransportStream) SetHeader(md metadata.MD) error {
	f.header = metadata.Join(f.header, md)
	return nil
}

func (f *fakeServerTransportStream) SendHeader(md metadata.MD) error { return f.SetHeader(md) }

func (f *fakeServerTransportStream) SetTrailer(md metadata.MD) error {
	f.trailer = metadata.Join(f.trailer, md)
	return nil
}

func TestIsIdempotentReplayed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		md       metadata.MD
		expected bool
	}{
		{
			name:     "header set to true",
			md:       metadata.Pairs(IdempotentReplayedHeader, IdempotentReplayedHeaderValue),
			expected: true,
		},
		{
			name:     "header set to another value",
			md:       metadata.Pairs(IdempotentReplayedHeader, "false"),
			expected: false,
		},
		{
			name:     "header absent",
			md:       metadata.New(nil),
			expected: false,
		},
		{
			name:     "nil metadata",
			md:       nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsIdempotentReplayed(tt.md); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestWithIdempotencyTracking_Finalize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		handler      func(ctx context.Context) error
		wantReplayed bool
	}{
		{
			name:         "handler that never replays sets no header",
			handler:      func(ctx context.Context) error { return nil },
			wantReplayed: false,
		},
		{
			name: "replayed success sets the header",
			handler: func(ctx context.Context) error {
				appctx.MarkIdempotencyReplayed(ctx)
				return nil
			},
			wantReplayed: true,
		},
		{
			// A cached 4xx is still a replay: the client must be able to tell that
			// its retry did not re-execute the request.
			name: "replayed error sets the header",
			handler: func(ctx context.Context) error {
				appctx.MarkIdempotencyReplayed(ctx)
				return errors.New("cached failure")
			},
			wantReplayed: true,
		},
		{
			name: "panicking handler still finalizes",
			handler: func(ctx context.Context) error {
				appctx.MarkIdempotencyReplayed(ctx)
				panic("boom")
			},
			wantReplayed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := &fakeServerTransportStream{}
			baseCtx := grpc.NewContextWithServerTransportStream(context.Background(), stream)

			func() {
				defer func() { _ = recover() }()
				ctx, finalize := WithIdempotencyTracking(baseCtx)
				defer finalize()

				if _, ok := appctx.GetIdempotencyResponseMetadata(ctx); !ok {
					t.Error("expected response metadata in the handler context")
				}
				_ = tt.handler(ctx)
			}()

			if got := IsIdempotentReplayed(stream.header); got != tt.wantReplayed {
				t.Errorf("expected replayed header %v, got %v (header=%v)", tt.wantReplayed, got, stream.header)
			}
		})
	}
}

func TestSetIdempotentReplayed_WithoutServerStream(t *testing.T) {
	t.Parallel()
	// Outside a gRPC handler there is no stream to write to; the call must be a
	// no-op rather than a panic.
	SetIdempotentReplayed(context.Background())
}
