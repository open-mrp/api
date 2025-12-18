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

	queries := sqlc.New(db)
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

	serverOpts := append(
		tracing.WithTracingInterceptors(),
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
