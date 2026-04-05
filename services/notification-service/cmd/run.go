package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/email"
	"github.com/augno/api/services/notification-service/internal/event"
	notificationgrpc "github.com/augno/api/services/notification-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/notification-service/internal/infrastructure/repository"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/notification-service/internal/service"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

func Run(
	ctx context.Context,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := new(config).withDefaults(getenv)
	if err := cfg.validate(); err != nil {
		return err
	}

	logger := slog.New(slog.NewTextHandler(stdout, nil))

	tracerShutdown, err := tracing.InitProvider(ctx, domain.ServiceName, getenv)
	if err != nil {
		return err
	}
	defer tracing.DeferShutdown(tracerShutdown)()

	workerTracer, err := tracing.NewWorkerTracerProvider(ctx, domain.ServiceName, getenv)
	if err != nil {
		return err
	}
	defer workerTracer.DeferClose()()

	db, err := db.NewDbPool(&db.Config{DBURI: cfg.DBURL})
	if err != nil {
		return err
	}
	defer db.Close()

	rabbitmq, err := messaging.NewRabbitMQ(ctx, &messaging.RabbitMQConfig{URI: cfg.RabbitMQURI})
	if err != nil {
		return err
	}
	defer rabbitmq.Close()

	queries := sqlc.New(db)

	templateRenderer, apiErr := email.NewTemplateRenderer()
	if apiErr != nil {
		return apiErr
	}

	notificationConfig, apiErr := new(service.NotificationSvcConfig).WithDefaults(queries, cfg.PlatformMode, cfg.AWSRegion, templateRenderer)
	if apiErr != nil {
		return apiErr
	}
	notificationSvc := service.NewNotificationSvc(notificationConfig)

	inboxRepo := repository.NewInboxRepo(queries)
	inboxPurgerRepo := repository.NewInboxPurgerRepo(queries)
	inboxPurger, err := messaging.NewInboxPurger(&messaging.InboxPurgerConfig{}, inboxPurgerRepo)
	if err != nil {
		return err
	}
	if err := inboxPurger.Start(ctx); err != nil {
		return err
	}
	defer inboxPurger.Stop()

	consumerTracer := workerTracer.Tracer(domain.ServiceName + ".consumer")
	notificationConsumer := event.NewNotificationConsumer(rabbitmq, notificationSvc, inboxRepo, templateRenderer, consumerTracer)
	if err := notificationConsumer.Listen(ctx); err != nil {
		return err
	}

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	notificationgrpc.NewGRPCHandler(server.Server(), notificationSvc)

	logger.Info("Notification service started", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
