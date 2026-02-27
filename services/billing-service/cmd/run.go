package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/services/billing-service/internal/event"
	"github.com/augno/api/services/billing-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/billing-service/internal/infrastructure/repository"
	"github.com/augno/api/services/billing-service/internal/infrastructure/sqlc"
	stripeinfra "github.com/augno/api/services/billing-service/internal/infrastructure/stripe"
	"github.com/augno/api/services/billing-service/internal/service"
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

	queries, err := sqlc.Prepare(ctx, dbpool)
	if err != nil {
		return err
	}
	defer queries.Close()

	outboxRepo := repository.NewOutboxEnqueuerRepo(dbpool, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName}, outboxRepo, rabbitmq)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	stripeClient := stripeinfra.NewStripeClient(&stripeinfra.ClientConfig{
		WebhookSecret: cfg.StripeWebhookSecret,
		APIKey:        cfg.StripeAPIKey,
	})

	coreClient, err := grpc.NewBillingCoreClient(cfg.CoreServiceURL)
	if err != nil {
		return err
	}
	defer coreClient.Close()

	if err := coreClient.WaitForReady(ctx); err != nil {
		return err
	}

	notificationClient, err := grpc.NewBillingNotificationClient(cfg.NotificationServiceURL)
	if err != nil {
		return err
	}
	defer notificationClient.Close()

	if err := notificationClient.WaitForReady(ctx); err != nil {
		return err
	}

	repoFactory := repository.NewRepoFactory(queries)

	billingSvc := service.NewBillingSvc(&service.BillingSvcConfig{
		Repos:              repoFactory,
		StripeClient:       stripeClient,
		CoreClient:         coreClient,
		FrontendURL:        cfg.FrontendURL,
		NotificationClient: notificationClient,
	})

	checkoutSvc := service.NewCheckoutSvc(&service.CheckoutSvcConfig{
		StripeClient:   stripeClient,
		BillingSvc:     billingSvc,
		PublishableKey: cfg.StripePublishableKey,
	})

	stripeWebhookSvc := service.NewStripeWebhookSvc(&service.StripeWebhookSvcConfig{
		Repos:        repoFactory,
		StripeClient: stripeClient,
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

	stripeEventLogRepo := repository.NewStripeEventLogRepo(queries)
	stripeWebhookConsumer := event.NewStripeWebhookConsumer(rabbitmq, inboxRepo, stripeEventLogRepo, coreClient, stripeClient)
	if err := stripeWebhookConsumer.Listen(ctx); err != nil {
		return err
	}

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	grpc.NewBillingHandler(server.Server(), billingSvc, stripeWebhookSvc, checkoutSvc)

	logger.Info("Billing service starting", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
