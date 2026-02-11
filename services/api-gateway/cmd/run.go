package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/infrastructure/publisher"
	"github.com/augno/api/services/api-gateway/internal/infrastructure/repository"
	"github.com/augno/api/services/api-gateway/internal/infrastructure/sqlc"
	"github.com/augno/api/services/api-gateway/internal/router"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

const (
	httpIdleTimeout     = time.Minute
	httpReadTimeout     = 5 * time.Second
	httpWriteTimeout    = 10 * time.Second
	httpShutdownTimeout = 10 * time.Second
)

// Run is the entry point for the API gateway. It initializes the necessary
// components and starts the HTTP server. We separate this out from `main` to
// make it easier to test.
func Run(
	ctx context.Context,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	// Create a context that is notified of interrupt and termination signals.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load the configuration and validate it.
	cfg := new(config).withDefaults(getenv)
	if err := cfg.validate(); err != nil {
		return err
	}

	// Initialize the tracing provider.
	tracerShutdown, err := tracing.InitProvider(ctx, domain.ServiceName, getenv)
	if err != nil {
		return err
	}
	defer tracing.DeferShutdown(tracerShutdown)()

	// Initialize the logger.
	logger := slog.New(slog.NewTextHandler(stdout, nil))

	// Initialize the database pool.
	dbPool, err := db.NewDbPool(&db.Config{DBURI: cfg.DBURI})
	if err != nil {
		return err
	}
	defer dbPool.Close()

	// Initialize the RabbitMQ client.
	rabbitmq, err := messaging.NewRabbitMQ(ctx, &messaging.RabbitMQConfig{URI: cfg.RabbitMQURI})
	if err != nil {
		return err
	}
	defer rabbitmq.Close()

	// Initialize the SQLC queries.
	queries, err := sqlc.Prepare(ctx, dbPool)
	if err != nil {
		return err
	}
	defer queries.Close()

	// Initialize the outbox enqueuer.
	outboxEnqueuerRepo := repository.NewOutboxEnqueuerRepo(dbPool, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName}, outboxEnqueuerRepo, rabbitmq)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	// Initialize the gRPC clients.

	// Auth Service
	authClient, err := grpcclient.NewAuthServiceClientWithURL(cfg.AuthServiceURI)
	if err != nil {
		return err
	}
	defer authClient.Close()

	logger.Info("Waiting for Auth Service to be ready...")
	if err := authClient.WaitForReady(ctx); err != nil {
		return err
	}

	// Core Service
	coreClient, err := grpcclient.NewCoreServiceClientWithURL(cfg.CoreServiceURI)
	if err != nil {
		return err
	}
	defer coreClient.Close()

	logger.Info("Waiting for Core Service to be ready...")
	if err := coreClient.WaitForReady(ctx); err != nil {
		return err
	}

	// Platform Service
	platformClient, err := grpcclient.NewPlatformServiceClientWithURL(cfg.PlatformServiceURI)
	if err != nil {
		return err
	}
	defer platformClient.Close()

	logger.Info("Waiting for Platform Service to be ready...")
	if err := platformClient.WaitForReady(ctx); err != nil {
		return err
	}

	// Initialize the request log publisher.
	reqLogPublisher := publisher.NewRequestLogOutboxPublisher(repository.NewOutboxRepo(queries))

	// Initialize the main router.
	mainBaseCfg := router.BuildBaseConfig(cfg.PlatformMode, "main ", authClient, coreClient, platformClient, reqLogPublisher, stdout)
	mainRouter := router.NewMainRouter(mainBaseCfg)

	// Initialize the auth router.
	authBaseCfg := router.BuildBaseConfig(cfg.PlatformMode, "auth ", authClient, coreClient, platformClient, reqLogPublisher, stdout)
	authRouter := router.NewAuthRouter(authBaseCfg)

	// Initialize the HTTP server.
	mux := http.NewServeMux()
	mux.Handle("/v1/auth/", authRouter)
	mux.Handle("/", mainRouter)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		IdleTimeout:  httpIdleTimeout,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Start the HTTP server.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received", "addr", server.Addr)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("error shutting down http server", "err", err)
			return fmt.Errorf("shutdown http server: %w", err)
		}
		logger.Info("server shutdown completed gracefully")

		if err := <-serverErr; err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server exit error: %w", err)
		}

	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("listen and serve: %w", err)
		}
	}

	return nil
}
