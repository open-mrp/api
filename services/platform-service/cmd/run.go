package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/event"
	"github.com/augno/api/services/platform-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/platform-service/internal/infrastructure/repository"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/platform-service/internal/service"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/pagination"
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

	pagination.Init(cfg.CursorHMACKey)

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

	dbpool, err := db.NewDbPool(&db.Config{DBURI: cfg.DBURL})
	if err != nil {
		return err
	}
	defer dbpool.Close()

	rabbitmq, err := messaging.NewRabbitMQ(ctx, &messaging.RabbitMQConfig{URI: cfg.RabbitMQURI})
	if err != nil {
		return err
	}
	defer rabbitmq.Close()

	queries := sqlc.New(dbpool)

	loggingSvc := service.NewLoggingSvc(&service.LoggingSvcConfig{
		Repos: repository.NewRepoFactory(queries),
	})

	auditSvc := service.NewAuditEventSvc(&service.AuditEventSvcConfig{
		Repos: repository.NewRepoFactory(queries),
	})

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

	consumerTracer := workerTracer.Tracer(domain.ServiceName + ".request_log_consumer")
	consumer := event.NewRequestLogConsumer(rabbitmq, loggingSvc, inboxRepo, consumerTracer)
	if err := consumer.Listen(ctx); err != nil {
		return err
	}

	auditConsumerTracer := workerTracer.Tracer(domain.ServiceName + ".audit_event_consumer")
	auditConsumer := event.NewAuditEventConsumer(rabbitmq, auditSvc, inboxRepo, auditConsumerTracer)
	if err := auditConsumer.Listen(ctx); err != nil {
		return err
	}

	// Start the outbox enqueuer to publish messages from the outbox table
	outboxRepo := repository.NewOutboxEnqueuerRepo(dbpool, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName}, outboxRepo, rabbitmq)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	idempotencyRepo := repository.NewIdempotencyKeyRepo(dbpool, queries)

	// Start the idempotency key cleanup worker to delete expired keys
	cleanupRepo := repository.NewCleanupRepo(queries)
	cleanupWorker, err := messaging.NewCleanupWorker(&messaging.CleanupConfig{}, cleanupRepo)
	if err != nil {
		return err
	}
	if err := cleanupWorker.Start(ctx); err != nil {
		return err
	}
	defer cleanupWorker.Stop()

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	grpc.NewGRPCHandler(server.Server(), idempotencyRepo)
	grpc.NewLoggingHandler(server.Server(), loggingSvc)
	grpc.NewAuditHandler(server.Server(), auditSvc)

	logger.Info("Platform service starting", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
