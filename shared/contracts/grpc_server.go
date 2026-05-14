package contracts

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/augno/api/shared/logging"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

const (
	// GRPCPort is the standard port all backend gRPC servers listen on inside the cluster.
	GRPCPort                           = 9092
	defaultServerMaxConnectionIdle     = 15 * time.Minute
	defaultServerMaxConnectionAge      = 30 * time.Minute
	defaultServerMaxConnectionAgeGrace = 5 * time.Second
	defaultServerKeepaliveTime         = 30 * time.Second
	defaultServerKeepaliveTimeout      = 5 * time.Second
	defaultServerMinPingTime           = 10 * time.Second
	defaultGracefulStopTimeout         = 5 * time.Second
	// defaultMaxRecvMsgSize is the maximum message size the server can receive (100 MB)
	defaultMaxRecvMsgSize = 100 * 1024 * 1024
	// defaultMaxSendMsgSize is the maximum message size the server can send (100 MB)
	defaultMaxSendMsgSize = 100 * 1024 * 1024
)

// GRPCServerConfig holds settings for a gRPC server.
type GRPCServerConfig struct {
	// KeepaliveParams (optional; default: 15m idle, 30m age, 5s grace, 30s ping, 5s timeout)
	// controls how the server manages idle connections, connection age limits, and
	// server-side ping behavior.
	KeepaliveParams keepalive.ServerParameters

	// EnforcementPolicy (optional; default: 10s min ping time, permit without stream)
	// controls the minimum time between client pings and whether pings are allowed
	// when there are no active streams.
	EnforcementPolicy keepalive.EnforcementPolicy

	// UnaryInterceptors (optional; default: SpanRenamer, Recovery, Identity, IdempotencyKey,
	// RequestID, ClientIP, CanonicalLog) is the chain of server-side unary interceptors.
	UnaryInterceptors []grpc.UnaryServerInterceptor
}

// WithDefaults fills zero-value fields with production defaults and returns a new config.
// When logger is nil, a default text handler logger writing to stdout is used.
func (c *GRPCServerConfig) WithDefaults(logger *slog.Logger) *GRPCServerConfig {
	if c == nil {
		c = &GRPCServerConfig{}
	}

	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	if len(c.UnaryInterceptors) == 0 {
		c.UnaryInterceptors = []grpc.UnaryServerInterceptor{
			tracing.UnarySpanRenamer(),
			RecoveryUnaryInterceptor(),
			IdentityUnaryServerInterceptor(),
			IdempotencyKeyUnaryServerInterceptor(),
			RequestIDUnaryServerInterceptor(),
			ClientIPUnaryServerInterceptor(),
			logging.CanonicalLogInterceptor(logger),
		}
	}

	return &GRPCServerConfig{
		KeepaliveParams: keepalive.ServerParameters{
			MaxConnectionIdle:     cmp.Or(c.KeepaliveParams.MaxConnectionIdle, defaultServerMaxConnectionIdle),
			MaxConnectionAge:      cmp.Or(c.KeepaliveParams.MaxConnectionAge, defaultServerMaxConnectionAge),
			MaxConnectionAgeGrace: cmp.Or(c.KeepaliveParams.MaxConnectionAgeGrace, defaultServerMaxConnectionAgeGrace),
			Time:                  cmp.Or(c.KeepaliveParams.Time, defaultServerKeepaliveTime),
			Timeout:               cmp.Or(c.KeepaliveParams.Timeout, defaultServerKeepaliveTimeout),
		},
		EnforcementPolicy: keepalive.EnforcementPolicy{
			MinTime:             cmp.Or(c.EnforcementPolicy.MinTime, defaultServerMinPingTime),
			PermitWithoutStream: true,
		},
		UnaryInterceptors: c.UnaryInterceptors,
	}
}

// validate checks that keepalive and enforcement parameters are sensible.
func (c *GRPCServerConfig) validate() error {
	if c.KeepaliveParams.Time < 0 {
		return errors.New("gRPC server: keepalive time must be non-negative")
	}
	if c.KeepaliveParams.Timeout < 0 {
		return errors.New("gRPC server: keepalive timeout must be non-negative")
	}
	if c.EnforcementPolicy.MinTime < 0 {
		return errors.New("gRPC server: enforcement min time must be non-negative")
	}
	return nil
}

// GRPCServer wraps a grpc.Server and its health server.
type GRPCServer struct {
	name         string
	server       *grpc.Server
	healthServer *health.Server
}

// NewGRPCServer creates a ready-to-use gRPC server with tracing, keepalive,
// enforcement policy, interceptor chain, and health checking pre-configured.
func NewGRPCServer(serverName string, logger *slog.Logger, config *GRPCServerConfig) (*GRPCServer, error) {
	config = config.WithDefaults(logger)
	if err := config.validate(); err != nil {
		return nil, err
	}

	serverOpts := append(
		tracing.WithTracingInterceptors(),
		grpc.KeepaliveParams(config.KeepaliveParams),
		grpc.KeepaliveEnforcementPolicy(config.EnforcementPolicy),
		grpc.ChainUnaryInterceptor(config.UnaryInterceptors...),
		grpc.MaxRecvMsgSize(defaultMaxRecvMsgSize),
		grpc.MaxSendMsgSize(defaultMaxSendMsgSize),
	)

	srv := grpc.NewServer(serverOpts...)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus(serverName, grpc_health_v1.HealthCheckResponse_SERVING)

	return &GRPCServer{name: serverName, server: srv, healthServer: healthSrv}, nil
}

// Server returns the underlying grpc.Server for service handler registration.
func (s *GRPCServer) Server() *grpc.Server {
	return s.server
}

// Serve listens on the given port and blocks until ctx is cancelled or a fatal
// serve error occurs. On cancellation it attempts a graceful stop with a timeout
// before forcing an immediate stop.
func (s *GRPCServer) Serve(ctx context.Context, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("gRPC server: failed to listen on port %d: %w", port, err)
	}

	errCh := make(chan error, 1)
	go func() {
		if serveErr := s.server.Serve(lis); serveErr != nil {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		stopped := make(chan struct{})
		go func() {
			s.server.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(defaultGracefulStopTimeout):
			s.server.Stop()
		}
		return nil
	case err := <-errCh:
		return err
	}
}
