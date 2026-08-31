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

	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	httptransport "github.com/open-mrp/api/services/api-gateway/internal/http"
	"github.com/open-mrp/api/services/api-gateway/internal/infrastructure/publisher"
	"github.com/open-mrp/api/services/api-gateway/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/api-gateway/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"

	// resourceregistry's init() registers the resourcekit Definitions for every resource the include resolver handles. Blank-imported so the init runs at startup; the loaders rely on SetCoreClient being called (a few lines below).
	_ "github.com/open-mrp/api/services/api-gateway/internal/resourceregistry"
	"github.com/open-mrp/api/services/api-gateway/internal/router"

	// versiontransforms's init() registers the version.Transformer chain that downgrades responses (and upgrades requests) for callers on older API versions. Blank-imported so the init runs at startup.
	_ "github.com/open-mrp/api/services/api-gateway/internal/versiontransforms"
	"github.com/open-mrp/api/services/api-gateway/internal/ws"
	"github.com/open-mrp/api/shared/db"
	"github.com/open-mrp/api/shared/lease"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

const (
	httpIdleTimeout     = time.Minute
	httpReadTimeout     = 10 * time.Second
	httpWriteTimeout    = 30 * time.Second
	httpShutdownTimeout = 10 * time.Second
)

// Run is the entry point for the API gateway. It initializes the necessary components and starts the HTTP server. We separate this out from `main` to make it easier to test.
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

	// Configure the frontend URL for request log links in error responses.
	httptransport.SetFrontendURL(cfg.FrontendURL)

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
	queries := sqlc.New(dbPool)

	leaseSvc := lease.New(repository.NewLeaseRepo(queries))

	// Initialize the outbox enqueuer.
	outboxEnqueuerRepo := repository.NewOutboxEnqueuerRepo(dbPool, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode}, outboxEnqueuerRepo, rabbitmq, leaseSvc)
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

	// Hand the core client to the resourcekit loaders. The resourceregistry package is blank-imported below to fire its init() definitions; the loaders rely on this client being set before any HTTP request runs.
	resourceloaders.SetCoreClient(coreClient.Client)
	resourceloaders.SetCoreSalesClient(coreClient.Sales)
	resourceloaders.SetCorePurchaseClient(coreClient.Purchase)
	resourceloaders.SetFulfillmentClient(coreClient.Fulfillment)
	resourceloaders.SetCorePickingClient(coreClient.Picking)
	resourceloaders.SetMachineDowntimeClient(coreClient.MachineDowntime)
	resourceloaders.SetDemandOverrideClient(coreClient.DemandOverride)
	resourceloaders.SetPortalDomainClient(coreClient.PortalDomain)
	resourceloaders.SetCoreShippingClient(coreClient.Shipping)
	resourceloaders.SetCoreReceivingClient(coreClient.Receiving)
	resourceloaders.SetCoreProductionRunClient(coreClient.ProductionRun)
	resourceloaders.SetAuthClient(authClient.Client)

	// Billing Service
	billingClient, err := grpcclient.NewBillingServiceClientWithURL(cfg.BillingServiceURI)
	if err != nil {
		return err
	}
	defer billingClient.Close()

	logger.Info("Waiting for Billing Service to be ready...")
	if err := billingClient.WaitForReady(ctx); err != nil {
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

	// Agent Service
	agentClient, err := grpcclient.NewAgentServiceClientWithURL(cfg.AgentServiceURI)
	if err != nil {
		return err
	}
	defer agentClient.Close()

	logger.Info("Waiting for Agent Service to be ready...")
	if err := agentClient.WaitForReady(ctx); err != nil {
		return err
	}

	// Notification Service
	notificationClient, err := grpcclient.NewNotificationServiceClientWithURL(cfg.NotificationServiceURI)
	if err != nil {
		return err
	}
	defer notificationClient.Close()

	logger.Info("Waiting for Notification Service to be ready...")
	if err := notificationClient.WaitForReady(ctx); err != nil {
		return err
	}

	// Wire the agent client into the resourcekit loaders for agent-* resources.
	resourceloaders.SetAgentClient(agentClient.Client)

	// Wire the logging client into the resourcekit loaders for request-log resources.
	resourceloaders.SetLoggingClient(platformClient.LoggingClient)

	// Wire the audit client into the resourcekit loaders for created_by resolution.
	resourceloaders.SetAuditClient(platformClient.AuditClient)

	// Wire the chat client into the resourcekit loaders for a message's expandable conversation / reply_to references.
	resourceloaders.SetChatClient(notificationClient.ChatClient)

	// Wire the email bridge client into the resourcekit loaders for an email inbox's expandable email_domain reference.
	resourceloaders.SetEmailBridgeClient(notificationClient.EmailBridgeClient)

	// Initialize the request log publisher.
	reqLogPublisher := publisher.NewRequestLogOutboxPublisher(repository.NewOutboxRepo(queries), coreClient.Client, cfg.FrontendURL, cfg.PlatformMode)

	// Initialize the main router.
	mainBaseCfg := router.BuildBaseConfig(cfg.PlatformMode, "main ", authClient, coreClient, billingClient, platformClient, agentClient, notificationClient, reqLogPublisher, stdout, cfg.TrustedProxyHops)
	mainRouter := router.NewMainRouter(mainBaseCfg)

	// Initialize the auth router.
	authBaseCfg := router.BuildBaseConfig(cfg.PlatformMode, "auth ", authClient, coreClient, billingClient, platformClient, agentClient, notificationClient, reqLogPublisher, stdout, cfg.TrustedProxyHops)
	authRouter := router.NewAuthRouter(authBaseCfg)

	// Initialize the webhook router (no auth, minimal middleware).
	webhookBaseCfg := router.BuildBaseConfig(cfg.PlatformMode, "webhook ", authClient, coreClient, billingClient, platformClient, agentClient, notificationClient, reqLogPublisher, stdout, cfg.TrustedProxyHops)
	webhookRouter := router.NewWebhookRouter(webhookBaseCfg)

	// Initialize WebSocket hub and event consumer.
	wsHub := ws.NewHub()
	if err := ws.StartEventConsumer(ctx, rabbitmq, wsHub); err != nil {
		return err
	}
	if err := ws.StartRunCompletedConsumer(ctx, rabbitmq, wsHub); err != nil {
		return err
	}
	if err := ws.StartNotificationConsumer(ctx, rabbitmq, wsHub); err != nil {
		return err
	}

	// Initialize the HTTP server.
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/ws", ws.NewHandler(wsHub, authClient, notificationClient, []byte(cfg.WSTicketSecret)))
	mux.HandleFunc("/v1/ws/ticket", ws.NewTicketHandler(authClient, []byte(cfg.WSTicketSecret)))
	mux.Handle("/v1/webhooks/", webhookRouter)
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

	// Initialize the trusted internal listener (agent traffic). Only started when a service token is configured. It must never be exposed behind the public ALB.
	var internalServer *http.Server
	if cfg.InternalServiceToken != "" {
		internalBaseCfg := router.BuildBaseConfig(cfg.PlatformMode, "internal ", authClient, coreClient, billingClient, platformClient, agentClient, notificationClient, reqLogPublisher, stdout, cfg.TrustedProxyHops)
		internalRouter := router.NewInternalRouter(internalBaseCfg, cfg.InternalServiceToken)
		internalMux := http.NewServeMux()
		internalMux.Handle("/", internalRouter)
		internalServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.InternalPort),
			Handler:      internalMux,
			IdleTimeout:  httpIdleTimeout,
			ReadTimeout:  httpReadTimeout,
			WriteTimeout: httpWriteTimeout,
			ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		}
	}

	// Start the HTTP server.
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	if internalServer != nil {
		go func() {
			logger.Info("internal server starting", "addr", internalServer.Addr)
			serverErr <- internalServer.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received", "addr", server.Addr)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		if internalServer != nil {
			if err := internalServer.Shutdown(shutdownCtx); err != nil {
				logger.Error("error shutting down internal http server", "err", err)
			}
		}
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
