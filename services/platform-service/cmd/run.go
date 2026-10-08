package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/event"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/grpc"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/llm"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/platform-service/internal/service"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/lease"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
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

	dbpool, err := db.NewDbPool(&db.Config{DBURI: cfg.DBURL, Application: domain.ServiceName})
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

	leaseSvc := lease.New(repository.NewLeaseRepo(queries))

	payloads, err := blobstore.Open(ctx, cfg.AWSRegion, cfg.PayloadsBucket)
	if err != nil {
		return err
	}
	repos := repository.NewRepoFactory(queries, payloads)

	loggingSvc := service.NewLoggingSvc(&service.LoggingSvcConfig{
		Repos: repos,
	})

	auditSvc := service.NewAuditEventSvc(&service.AuditEventSvcConfig{
		Repos: repos,
	})

	inboxRepo := repository.NewInboxRepo(queries)
	inboxPurgerRepo := repository.NewInboxPurgerRepo(queries)
	inboxPurger, err := messaging.NewInboxPurger(&messaging.InboxPurgerConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode}, inboxPurgerRepo, leaseSvc)
	if err != nil {
		return err
	}
	if err := inboxPurger.Start(ctx); err != nil {
		return err
	}
	defer inboxPurger.Stop()

	if _, err := messaging.RegisterInboxGauges(domain.ServiceName, repository.NewInboxStatsRepo(queries)); err != nil {
		return err
	}

	consumerTracer := workerTracer.Tracer(domain.ServiceName + ".request_log_consumer")
	consumer := event.NewRequestLogConsumer(rabbitmq, loggingSvc, consumerTracer)
	if err := consumer.Listen(ctx); err != nil {
		return err
	}

	auditConsumerTracer := workerTracer.Tracer(domain.ServiceName + ".audit_event_consumer")
	auditConsumer := event.NewAuditEventConsumer(rabbitmq, auditSvc, inboxRepo, auditConsumerTracer)
	if err := auditConsumer.Listen(ctx); err != nil {
		return err
	}

	// Account follow-ups: registrations are always recorded; drafting runs only when enabled.
	var followupDrafter domain.AccountFollowupDrafter
	if cfg.AccountFollowupEnabled {
		followupDrafter, err = llm.NewAccountFollowupDrafter(&llm.AccountFollowupDrafterConfig{
			StripeSecretKey: cfg.StripeSecretKey,
			Model:           cfg.AccountFollowupModel,
		})
		if err != nil {
			return err
		}
	}
	followupSvc, err := service.NewAccountFollowupSvc(&service.AccountFollowupSvcConfig{
		Repos:           repos,
		Tx:              service.NewTransactionManager(dbpool, queries, payloads),
		ReviewerEmail:   cfg.AccountFollowupReviewerEmail,
		ReviewBaseURL:   cfg.AccountFollowupReviewBaseURL,
		Drafter:         followupDrafter,
		Delay:           cfg.AccountFollowupDelay,
		ExcludedDomains: cfg.AccountFollowupExcludedDomains,
	})
	if err != nil {
		return err
	}
	followupConsumer := event.NewAccountFollowupConsumer(rabbitmq, followupSvc, inboxRepo, workerTracer.Tracer(domain.ServiceName+".account_followup_consumer"))
	if err := followupConsumer.ListenSchedule(ctx); err != nil {
		return err
	}
	if cfg.AccountFollowupEnabled {
		if err := followupConsumer.ListenDraft(ctx); err != nil {
			return err
		}
		followupScheduler, err := service.NewAccountFollowupScheduler(&service.AccountFollowupSchedulerConfig{
			Svc:          followupSvc,
			Lease:        leaseSvc,
			PollInterval: cfg.AccountFollowupPollInterval,
		})
		if err != nil {
			return err
		}
		followupScheduler.Start(ctx)
		defer followupScheduler.Stop()
	}

	// Start the outbox enqueuer to publish messages from the outbox table
	outboxRepo := repository.NewOutboxEnqueuerRepo(dbpool, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode}, outboxRepo, rabbitmq, leaseSvc)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	if _, err := messaging.RegisterOutboxGauges(domain.ServiceName, repository.NewOutboxStatsRepo(queries)); err != nil {
		return err
	}

	idempotencyRepo := repository.NewIdempotencyKeyRepo(dbpool, queries, payloads)

	if !cfg.BackfillsPaused {
		if err := startBackfills(ctx, cfg, queries, payloads, leaseSvc); err != nil {
			return err
		}
	}

	// Start the idempotency key cleanup worker to delete expired keys
	cleanupRepo := repository.NewCleanupRepo(queries)
	cleanupWorker, err := messaging.NewCleanupWorker(&messaging.CleanupConfig{}, cleanupRepo, leaseSvc)
	if err != nil {
		return err
	}
	if err := cleanupWorker.Start(ctx); err != nil {
		return err
	}
	defer cleanupWorker.Stop()

	// Start the message failure monitor to email an alert when async messages fail to process
	// (dead-lettered inbox handlers, exhausted outbox publishes, crash-stuck rows). It scans the
	// shared message_inbox/message_outbox tables that the whole MySQL fleet writes to.
	failureMonitor, err := messaging.NewFailureMonitor(
		&messaging.FailureMonitorConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode},
		repository.NewFailureMonitorRepo(queries),
		repository.NewOutboxRepo(queries),
		leaseSvc,
	)
	if err != nil {
		return err
	}
	if err := failureMonitor.Start(ctx); err != nil {
		return err
	}
	defer failureMonitor.Stop()

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	grpc.NewGRPCHandler(server.Server(), idempotencyRepo)
	grpc.NewLoggingHandler(server.Server(), loggingSvc)
	grpc.NewAuditHandler(server.Server(), auditSvc)
	grpc.NewAccountFollowupHandler(server.Server(), followupSvc)

	logger.Info("Platform service starting", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
