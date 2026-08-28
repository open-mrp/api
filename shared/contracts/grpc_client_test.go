package contracts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/keepalive"
)

func TestNewGRPCClientConn_InvalidTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		target  GRPCConnTarget
		config  *GRPCClientConfig
		wantErr error
	}{
		{
			name:    "empty url",
			target:  GRPCConnTarget{Name: "auth-service"},
			wantErr: errGRPCClientURLIsEmpty,
		},
		{
			name:    "empty name",
			target:  GRPCConnTarget{URL: "auth-service:9092"},
			wantErr: errGRPCClientTargetNameIsEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := NewGRPCClientConn(tt.target, tt.config)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if conn != nil {
				t.Error("expected nil connection")
			}
		})
	}
}

func TestNewGRPCClientConn_InvalidKeepalive(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config *GRPCClientConfig
	}{
		{
			name:   "negative keepalive time",
			config: &GRPCClientConfig{KeepaliveParams: keepalive.ClientParameters{Time: -1}},
		},
		{
			name:   "negative keepalive timeout",
			config: &GRPCClientConfig{KeepaliveParams: keepalive.ClientParameters{Timeout: -1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := NewGRPCClientConn(GRPCConnTarget{URL: "auth-service:9092", Name: "auth-service"}, tt.config)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if conn != nil {
				t.Error("expected nil connection")
			}
			if !strings.Contains(err.Error(), "must be non-negative") {
				t.Errorf("expected a keepalive validation error, got %v", err)
			}
		})
	}
}

func TestGRPCClientConn_WaitForReady_NoConnection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		conn *GRPCClientConn
	}{
		{
			name: "nil receiver",
			conn: nil,
		},
		{
			name: "nil underlying connection",
			conn: &GRPCClientConn{name: "auth-service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.conn.WaitForReady(context.Background()); !errors.Is(err, errGRPCClientConnectionNotEstablished) {
				t.Errorf("expected %v, got %v", errGRPCClientConnectionNotEstablished, err)
			}
			if err := tt.conn.Close(); !errors.Is(err, errGRPCClientConnectionNotEstablished) {
				t.Errorf("expected %v from Close, got %v", errGRPCClientConnectionNotEstablished, err)
			}
		})
	}
}

// A dependency whose health service never reports SERVING must not hang past the
// caller's deadline.
func TestGRPCClientConn_WaitForReady_ContextEnded(t *testing.T) {
	t.Parallel()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
	defer cancelExpired()

	tests := []struct {
		name    string
		ctx     context.Context
		wantErr error
	}{
		{
			name:    "canceled context",
			ctx:     canceled,
			wantErr: context.Canceled,
		},
		{
			name:    "expired deadline",
			ctx:     expired,
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// grpc.NewClient is lazy, so nothing is dialed: the unreachable address
			// only guarantees the health check can never report SERVING.
			conn, err := NewGRPCClientConn(GRPCConnTarget{URL: "passthrough:///127.0.0.1:9", Name: "auth-service"}, nil)
			if err != nil {
				t.Fatalf("expected a connection, got %v", err)
			}
			defer conn.Close()

			err = conn.WaitForReady(tt.ctx)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
