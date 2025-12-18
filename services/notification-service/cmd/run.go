package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/augno/api/services/notification-service/internal/event"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/notification-service/internal/service"
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

	tracerShutdown, err := tracing.InitProvider("notification-service")
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
	notificationSvc, apiErr := service.NewDefaultNotificationSvc(queries, cfg.AWSRegion)
	if apiErr != nil {
		return fmt.Errorf("failed to initialize notification service: %w", apiErr)
	}

	notificationConsumer := event.NewNotificationConsumer(rabbitmq, notificationSvc)
	if err := notificationConsumer.Listen(); err != nil {
		return fmt.Errorf("failed to start notification consumer: %w", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Shutting down notification service...")
	return nil
}
