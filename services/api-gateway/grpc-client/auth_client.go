package grpcclient

import (
	"context"
	"fmt"
	"time"

	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

type AuthServiceClient struct {
	Client pb.AuthServiceClient
	conn   *grpc.ClientConn
}

func NewAuthServiceClient(
	getenv func(string) string,
) (*AuthServiceClient, error) {
	authServiceURL := getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		return nil, fmt.Errorf("AUTH_SERVICE_URL is not set")
	}
	return NewAuthServiceClientWithURL(authServiceURL)
}

func NewAuthServiceClientWithURL(authServiceURL string) (*AuthServiceClient, error) {
	// Configure keepalive to maintain connections and detect dead connections quickly
	keepaliveParams := keepalive.ClientParameters{
		Time:                60 * time.Second, // Send keepalive pings every 60 seconds
		Timeout:             5 * time.Second,  // Wait 5 seconds for ping ack before considering connection dead
		PermitWithoutStream: false,            // Only send pings when there are active streams
	}

	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepaliveParams),
	)

	conn, err := grpc.NewClient(authServiceURL, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service at %s: %w", authServiceURL, err)
	}

	client := pb.NewAuthServiceClient(conn)

	return &AuthServiceClient{
		Client: client,
		conn:   conn,
	}, nil
}

func (c *AuthServiceClient) WaitForReady(ctx context.Context) error {
	healthClient := grpc_health_v1.NewHealthClient(c.conn)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
				return nil
			}
		}
	}
}

func (c *AuthServiceClient) Close() {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return
		}
	}
}
