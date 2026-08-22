package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-mrp/api/services/auth-service/internal/domain"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/grpc"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/auth-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/auth-service/internal/service"
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

	leaseSvc := lease.New(repository.NewLeaseRepo(queries))

	outboxRepo := repository.NewOutboxEnqueuerRepo(db, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode}, outboxRepo, rabbitmq, leaseSvc)
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

	// Billing service client (optional — only if billing service is configured)
	var billingClient domain.AuthBillingClient
	if cfg.BillingServiceURL != "" {
		bc, billingErr := grpc.NewAuthBillingClient(cfg.BillingServiceURL)
		if billingErr != nil {
			return billingErr
		}
		defer bc.Close()
		billingClient = bc

		logger.Info("Waiting for Billing Service to be ready...")
		if billingErr := billingClient.WaitForReady(ctx); billingErr != nil {
			return billingErr
		}
	}

	txManager := service.NewTransactionManager(db, queries)
	authConfig := service.BuildAuthSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient)
	authConfig.TxManager = txManager
	authSvc := service.NewAuthSvc(authConfig)

	userConfig := service.BuildUserSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient)
	userConfig.TxManager = txManager
	userSvc := service.NewUserSvc(userConfig)

	tokenConfig := service.BuildTokenSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient)
	tokenConfig.TxManager = txManager
	tokenSvc := service.NewTokenSvc(tokenConfig)

	passwordConfig := service.BuildPasswordSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient)
	passwordConfig.TxManager = txManager
	passwordSvc := service.NewPasswordSvc(passwordConfig)

	apiKeyConfig := service.BuildAPIKeySvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient, cfg.DocAPIKeyEncryptionKey)
	apiKeyConfig.TxManager = txManager
	apiKeySvc := service.NewAPIKeySvc(apiKeyConfig)

	docAPIKeyConfig := service.BuildDocAPIKeySvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient, cfg.DocAPIKeyEncryptionKey)
	docAPIKeyConfig.TxManager = txManager
	docAPIKeySvc := service.NewDocAPIKeySvc(docAPIKeyConfig)

	registrationSessionConfig := service.BuildRegistrationSessionSvcConfig(queries, cfg.JWTSecret, cfg.Pepper, cfg.FrontendURL, coreClient, billingClient)
	registrationSessionConfig.TxManager = txManager
	registrationSessionSvc := service.NewRegistrationSessionSvc(registrationSessionConfig)

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	grpc.NewGRPCHandler(server.Server(), authSvc, userSvc, tokenSvc, passwordSvc, apiKeySvc, docAPIKeySvc, registrationSessionSvc)

	logger.Info("Auth service started", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
