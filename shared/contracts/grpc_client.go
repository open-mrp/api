package contracts

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

var (
	errGRPCClientURLIsEmpty               = errors.New("gRPC client: URL is empty")
	errGRPCClientTargetNameIsEmpty        = errors.New("gRPC client: target name is empty")
	errGRPCClientConnectionNotEstablished = errors.New("gRPC client: connection is not established")
)

const (
	defaultKeepaliveTime                = 60 * time.Second
	defaultKeepaliveTimeout             = 5 * time.Second
	defaultKeepalivePermitWithoutStream = false
	defaultWaitForReadyInterval         = 100 * time.Millisecond
	// defaultMaxCallRecvMsgSize is the maximum message size the client can receive (100 MB)
	defaultMaxCallRecvMsgSize = 100 * 1024 * 1024
	// defaultMaxCallSendMsgSize is the maximum message size the client can send (100 MB)
	defaultMaxCallSendMsgSize = 100 * 1024 * 1024
)

// GRPCClientConn is an active connection to a gRPC server.
type GRPCClientConn struct {
	conn *grpc.ClientConn
	name string
}

// GRPCConnTarget identifies the gRPC server to connect to.
type GRPCConnTarget struct {
	// URL is the dial address (e.g. "auth-service:9092").
	URL string
	// Name is a human-readable identifier used in logs and error messages.
	Name string
}

// validate validates the gRPC connection target.
func (t *GRPCConnTarget) validate() error {
	if t.URL == "" {
		return errGRPCClientURLIsEmpty
	}
	if t.Name == "" {
		return errGRPCClientTargetNameIsEmpty
	}
	return nil
}

// GRPCClientConfig holds dial-time settings for a gRPC client connection.
type GRPCClientConfig struct {
	// KeepaliveParams (optional; default: 60s ping, 5s timeout) controls how often the
	// client pings the server and how long it waits for a response before considering
	// the connection dead.
	KeepaliveParams keepalive.ClientParameters

	// UnaryInterceptors (optional; default: retry-on-transient) is the chain of client-side
	// unary interceptors.
	UnaryInterceptors []grpc.UnaryClientInterceptor
}

// WithDefaults fills zero keepalive fields with defaults and returns a new options value.
func (c *GRPCClientConfig) WithDefaults(targetName string) *GRPCClientConfig {
	if c == nil {
		c = &GRPCClientConfig{}
	}

	if len(c.UnaryInterceptors) == 0 {
		c.UnaryInterceptors = []grpc.UnaryClientInterceptor{
			retryOnTransientUnaryClientInterceptor(targetName),
		}
	}

	return &GRPCClientConfig{
		KeepaliveParams: keepalive.ClientParameters{
			Time:                cmp.Or(c.KeepaliveParams.Time, defaultKeepaliveTime),
			Timeout:             cmp.Or(c.KeepaliveParams.Timeout, defaultKeepaliveTimeout),
			PermitWithoutStream: cmp.Or(c.KeepaliveParams.PermitWithoutStream, defaultKeepalivePermitWithoutStream),
		},
		UnaryInterceptors: c.UnaryInterceptors,
	}
}

// validate checks that keepalive parameters are sensible.
func (c *GRPCClientConfig) validate() error {
	if c.KeepaliveParams.Time < 0 {
		return errors.New("gRPC client: keepalive time must be non-negative")
	}
	if c.KeepaliveParams.Timeout < 0 {
		return errors.New("gRPC client: keepalive timeout must be non-negative")
	}
	return nil
}

// NewGRPCClientConn creates a new gRPC client connection. The connection is established
// using insecure credentials. Since all these connections are within the k8 cluster and
// not over a public network, this acceptable.
func NewGRPCClientConn(target GRPCConnTarget, config *GRPCClientConfig) (*GRPCClientConn, error) {
	if err := target.validate(); err != nil {
		return nil, err
	}

	config = config.WithDefaults(target.Name)
	if err := config.validate(); err != nil {
		return nil, err
	}

	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(config.KeepaliveParams),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(defaultMaxCallRecvMsgSize),
			grpc.MaxCallSendMsgSize(defaultMaxCallSendMsgSize),
		),
	)

	if len(config.UnaryInterceptors) > 0 {
		dialOptions = append(dialOptions, grpc.WithChainUnaryInterceptor(config.UnaryInterceptors...))
	}

	conn, err := grpc.NewClient(target.URL, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("gRPC client: failed to connect to %s at %s: %w", target.Name, target.URL, err)
	}

	return &GRPCClientConn{conn: conn, name: target.Name}, nil
}

// Conn returns the underlying gRPC client connection.
func (c *GRPCClientConn) Conn() *grpc.ClientConn {
	return c.conn
}

// WaitForReady waits for the gRPC server to be ready. If an error
// occurs, it is returned. If the context is canceled, the context
// error is returned. Otherwise, we return when the SERVING status
// is returned.
func (c *GRPCClientConn) WaitForReady(ctx context.Context) error {
	if c == nil {
		return errGRPCClientConnectionNotEstablished
	}
	if c.conn == nil {
		return errGRPCClientConnectionNotEstablished
	}

	healthClient := grpc_health_v1.NewHealthClient(c.conn)

	tickerCh := time.NewTicker(defaultWaitForReadyInterval)
	defer tickerCh.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tickerCh.C:
			resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
				return nil
			}
		}
	}
}

// Close closes the gRPC client connection.
func (c *GRPCClientConn) Close() error {
	if c == nil {
		return errGRPCClientConnectionNotEstablished
	}
	if c.conn == nil {
		return errGRPCClientConnectionNotEstablished
	}

	return c.conn.Close()
}
