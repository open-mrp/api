package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcserver "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"

	"github.com/augno/api/services/auth-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/auth-service/internal/service"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

func Run(
	ctx context.Context,
	getenv func(string) string,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := loadConfig(getenv)
	if err != nil {
		return err
	}

	tracerShutdown, err := tracing.InitProvider("auth-service")
	if err != nil {
		return fmt.Errorf("failed to initialize tracer: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracerShutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown tracer", "error", err)
		}
	}()

	db, err := db.NewDbPool(cfg.DBURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	slog.Info("Database connection established and warmed up.")

	rabbitmq, err := messaging.NewRabbitMQ(cfg.RabbitMQURI)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer rabbitmq.Close()
	if !rabbitmq.IsReady() {
		return fmt.Errorf("RabbitMQ connection is not ready")
	}
	slog.Info("RabbitMQ connection established and warmed up.")

	queries, err := sqlc.Prepare(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to prepare queries: %w", err)
	}
	defer queries.Close()

	txManager := service.NewTransactionManager(db, queries)
	authConfig := service.DefaultAuthSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, rabbitmq, cfg.TemplatesDir)
	authConfig.TxManager = txManager
	authSvc := service.NewAuthSvc(authConfig)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", cfg.Port, err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	_ = logger

	// Configure keepalive to manage idle connections and prevent "too_many_pings" errors
	kaServerParams := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Minute, // If a client is idle for 15 minutes, send a GOAWAY
		MaxConnectionAge:      30 * time.Minute, // If any connection is older than 30 minutes, send a GOAWAY
		MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before closing
		Time:                  30 * time.Second, // Ping the client every 30 seconds if it's idle
		Timeout:               5 * time.Second,  // Wait 5 seconds for ping ack before considering connection dead
	}

	kaPolicy := keepalive.EnforcementPolicy{
		MinTime:             10 * time.Second, // Minimum time between client pings
		PermitWithoutStream: true,             // Allow pings even if there are no active streams
	}

	serverOpts := append(
		tracing.WithTracingInterceptors(),
		grpcserver.KeepaliveParams(kaServerParams),
		grpcserver.KeepaliveEnforcementPolicy(kaPolicy),
		grpcserver.ChainUnaryInterceptor(
			tracing.UnarySpanRenamer(),
			contracts.RecoveryUnaryInterceptor(),
		),
	)
	grpcServer := grpcserver.NewServer(serverOpts...)
	grpc.NewGRPCHandler(grpcServer, authSvc)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	// Only set SERVING status after all dependencies (DB and RabbitMQ) are ready
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	errCh := make(chan error, 1)
	go func() {
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		stopped := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(5 * time.Second):
			grpcServer.Stop()
		}
		return nil
	case err := <-errCh:
		return err
	}
}
