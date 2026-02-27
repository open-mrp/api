package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/event"
	"github.com/augno/api/services/core-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/core-service/internal/infrastructure/repository"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/core-service/internal/mediator"
	"github.com/augno/api/services/core-service/internal/service"
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

	queries, err := sqlc.Prepare(ctx, db)
	if err != nil {
		return err
	}
	defer queries.Close()

	outboxEnqueuerRepo := repository.NewOutboxEnqueuerRepo(db, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName}, outboxEnqueuerRepo, rabbitmq)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	repoFactory := repository.NewRepoFactory(queries)
	txManager := service.NewTransactionManager(db, queries)

	mediatorFactory := mediator.NewMediatorFactory()
	accountSvc := service.NewAccountSvc(&service.AccountSvcConfig{RepoFactory: repoFactory, MediatorFactory: mediatorFactory, TxManager: txManager})
	sandboxSvc := service.NewSandboxSvc(&service.SandboxSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	unitSvc := service.NewUnitSvc(&service.UnitSvcConfig{
		Repos: repoFactory,
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

	purgeRepo := repository.NewPurgeRepo(db)
	purgeConsumer := event.NewPurgeConsumer(rabbitmq, inboxRepo, purgeRepo)
	if err := purgeConsumer.Listen(ctx); err != nil {
		return err
	}

	seeder := repository.NewSandboxSeeder(db)
	seedConsumer := event.NewSeedConsumer(rabbitmq, inboxRepo, seeder)
	if err := seedConsumer.Listen(ctx); err != nil {
		return err
	}

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	grpc.NewGRPCHandler(server.Server(), accountSvc, sandboxSvc, unitSvc)

	logger.Info("Core service started", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
