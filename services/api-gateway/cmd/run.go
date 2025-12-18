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
	"github.com/augno/api/services/api-gateway/internal/infrastructure/publisher"
	"github.com/augno/api/services/api-gateway/internal/router"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

const (
	tracerShutdownTimeout = 5 * time.Second
	httpIdleTimeout       = time.Minute
	httpReadTimeout       = 5 * time.Second
	httpWriteTimeout      = 10 * time.Second
	httpShutdownTimeout   = 10 * time.Second
)

// Run starts the API server with the given dependencies
func Run(
	ctx context.Context,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig(getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	tracerShutdown, err := tracing.InitProvider("api-gateway")
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, tracerShutdownTimeout)
		defer cancel()
		if err := tracerShutdown(shutdownCtx); err != nil {
			fmt.Fprintf(stderr, "error shutting down tracer: %s\n", err)
		}
	}()

	logger := slog.New(slog.NewTextHandler(stdout, nil))

	authClient, err := grpcclient.NewAuthServiceClientWithURL(cfg.AuthServiceURL)
	if err != nil {
		return fmt.Errorf("init auth client: %w", err)
	}
	defer authClient.Close()

	logger.Info("Waiting for Auth Service to be ready...")
	if err := authClient.WaitForReady(ctx); err != nil {
		return fmt.Errorf("wait for auth service ready: %w", err)
	}
	logger.Info("Auth Service (and Database) connection ready.")

	rabbitmq, err := messaging.NewRabbitMQ(cfg.RabbitMQURI)
	if err != nil {
		return fmt.Errorf("init rabbitmq: %w", err)
	}
	defer rabbitmq.Close()
	if !rabbitmq.IsReady() {
		return fmt.Errorf("RabbitMQ connection is not ready")
	}
	logger.Info("RabbitMQ connection established and warmed up.")

	reqLogRepo := publisher.NewRequestLogPublisher(rabbitmq)

	// Create main router for non-auth endpoints
	mainBaseCfg := router.BuildBaseConfig(cfg.PlatformMode, "main ", authClient, reqLogRepo, stdout)
	mainRouter := router.NewMainRouter(mainBaseCfg)

	// Create auth router for auth endpoints
	authBaseCfg := router.BuildBaseConfig(cfg.PlatformMode, "auth ", authClient, reqLogRepo, stdout)
	authRouter := router.NewAuthRouter(authBaseCfg)

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
		// Server died without a shutdown signal.
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("listen and serve: %w", err)
		}
	}

	return nil
}
