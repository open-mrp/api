package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/augno/api/services/auth-service/internal/domain"
	"github.com/augno/api/services/auth-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/auth-service/internal/infrastructure/repository"
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

	outboxRepo := repository.NewOutboxEnqueuerRepo(db, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName}, outboxRepo, rabbitmq)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	coreClient, apiErr := grpc.NewAuthCoreClient(cfg.CoreServiceURL)
	if apiErr != nil {
		return apiErr
	}
	defer coreClient.Close()

	logger.Info("Waiting for Core Service to be ready...")
	if apiErr := coreClient.WaitForReady(ctx); apiErr != nil {
		return apiErr
	}

	txManager := service.NewTransactionManager(db, queries)
	authConfig := service.DefaultAuthSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient)
	authConfig.TxManager = txManager
	authSvc := service.NewAuthSvc(authConfig)

	userConfig := service.DefaultUserSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient)
	userConfig.TxManager = txManager
	userSvc := service.NewUserSvc(userConfig)

	tokenConfig := service.DefaultTokenSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient)
	tokenConfig.TxManager = txManager
	tokenSvc := service.NewTokenSvc(tokenConfig)

	passwordConfig := service.DefaultPasswordSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient)
	passwordConfig.TxManager = txManager
	passwordSvc := service.NewPasswordSvc(passwordConfig)

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	grpc.NewGRPCHandler(server.Server(), authSvc, userSvc, tokenSvc, passwordSvc)

	logger.Info("Auth service started", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
