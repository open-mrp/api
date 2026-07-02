package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/augno/api/services/agent-service/internal/agents"
	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/services/agent-service/internal/event"
	agentdb "github.com/augno/api/services/agent-service/internal/infrastructure/db"
	agentgrpc "github.com/augno/api/services/agent-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/agent-service/internal/infrastructure/httpgateway"
	"github.com/augno/api/services/agent-service/internal/infrastructure/repository"
	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/agent-service/internal/infrastructure/stub"
	"github.com/augno/api/services/agent-service/internal/llm"
	"github.com/augno/api/services/agent-service/internal/mediator"
	"github.com/augno/api/services/agent-service/internal/service"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/lease"
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

	pgpool, err := agentdb.NewPgPool(ctx, cfg.DBURL)
	if err != nil {
		return err
	}
	defer pgpool.Close()

	rabbitmq, err := messaging.NewRabbitMQ(ctx, &messaging.RabbitMQConfig{URI: cfg.RabbitMQURI})
	if err != nil {
		return err
	}
	defer rabbitmq.Close()

	queries := sqlc.New(pgpool)

	leaseSvc := lease.New(repository.NewLeaseRepo(queries))

	outboxRepo := repository.NewOutboxEnqueuerRepo(pgpool, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{
		ServiceName:  domain.ServiceName,
		PlatformMode: cfg.PlatformMode,
		// Chat runs kick the enqueuer (OutboxNotifier below) so they start instantly; this tightens the
		// idle-backoff ceiling below the shared default so an un-kicked streaming event still posts
		// within 500ms rather than waiting out a longer idle poll.
		MaxPollInterval: 500 * time.Millisecond,
	}, outboxRepo, rabbitmq, leaseSvc)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	inboxPurgerRepo := repository.NewInboxPurgerRepo(queries)
	inboxPurger, err := messaging.NewInboxPurger(&messaging.InboxPurgerConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode}, inboxPurgerRepo, leaseSvc)
	if err != nil {
		return err
	}
	if err := inboxPurger.Start(ctx); err != nil {
		return err
	}
	defer inboxPurger.Stop()

	// Core-service gRPC client
	coreClient, err := agentgrpc.NewAgentCoreClient(cfg.CoreServiceURL)
	if err != nil {
		return err
	}
	defer coreClient.Close()

	if err := coreClient.WaitForReady(ctx); err != nil {
		return err
	}

	// Billing-service gRPC client (required for Stripe customer ID resolution)
	billingClient, err := agentgrpc.NewAgentBillingClient(cfg.BillingServiceURL)
	if err != nil {
		return err
	}
	defer billingClient.Close()

	if err := billingClient.WaitForReady(ctx); err != nil {
		return err
	}

	// --- Phase 2: LLM + Run Engine ---

	// LLM providers (stubbed in test mode)
	var providers map[string]llm.LLMProvider
	// Every provider routes through the single Stripe AI Gateway; the keys just mirror inferProvider.
	if cfg.PlatformMode.IsTest() {
		stubProvider := &stub.LLMProvider{}
		providers = map[string]llm.LLMProvider{
			"anthropic": stubProvider,
			"openai":    stubProvider,
			"google":    stubProvider,
			"xai":       stubProvider,
		}
	} else {
		// Anthropic models use the native Messages API (real thinking blocks + signatures for
		// reasoning streaming); the rest use the OpenAI-compatible /chat/completions surface. Both
		// route through the Stripe AI Gateway, so usage is metered per customer either way.
		gateway := llm.NewGatewayProvider(cfg.StripeSecretKey)
		anthropicNative := llm.NewAnthropicMessagesProvider(cfg.StripeSecretKey)
		providers = map[string]llm.LLMProvider{
			"anthropic": anthropicNative,
			"openai":    gateway,
			"google":    gateway,
			"xai":       gateway,
		}
	}

	// Gateway client for generated endpoint-tools (optional; only when configured).
	var gatewayClient domain.GatewayClient
	if cfg.GatewayInternalURL != "" && cfg.InternalServiceToken != "" {
		gatewayClient = httpgateway.NewClient(cfg.GatewayInternalURL, cfg.InternalServiceToken)
	}

	// Notification-service client for the agent's email reply/draft tools (optional).
	var notificationClient domain.NotificationClient
	if cfg.NotificationServiceURL != "" {
		nc, err := agentgrpc.NewAgentNotificationClient(cfg.NotificationServiceURL)
		if err != nil {
			return err
		}
		defer nc.Close()
		notificationClient = nc
	}

	// Repo factory
	repoFactory := repository.NewRepoFactory(queries)

	// Tool handler registry
	toolRegistry := agents.NewToolHandlerRegistry()
	agents.RegisterTools(toolRegistry)

	// Runner service
	runner := service.NewRunnerSvc(&service.RunnerConfig{
		Repos:              repoFactory,
		ToolRegistry:       toolRegistry,
		LLMProviders:       providers,
		OutboxRepo:         repoFactory.NewOutboxRepo(),
		CoreClient:         coreClient,
		GatewayClient:      gatewayClient,
		NotificationClient: notificationClient,
		Broker:             rabbitmq,
		BillingClient:      billingClient,
		OutboxNotifier:     enqueuer,
	})

	// Run consumer
	inboxRepo := repository.NewInboxRepo(queries)
	runConsumer := event.NewRunConsumer(rabbitmq, inboxRepo, runner)
	if err := runConsumer.Listen(ctx); err != nil {
		return err
	}
	if err := runConsumer.ListenContinueRun(ctx); err != nil {
		return err
	}

	// Plan gate adapter (uses core client to check plan code)
	planGate := agentgrpc.NewPlanGateAdapter(coreClient)

	// Scheduler
	scheduler := service.NewSchedulerSvc(&service.SchedulerConfig{
		Repos:      repoFactory,
		OutboxRepo: repoFactory.NewOutboxRepo(),
		PlanGate:   planGate,
		Lease:      leaseSvc,
	})
	if err := scheduler.Start(ctx); err != nil {
		return err
	}
	defer scheduler.Stop()

	// Agent definition service
	mediatorFactory := mediator.NewMediatorFactory()
	txManager := service.NewTransactionManager(pgpool, queries)
	agentDefSvc := service.NewAgentDefinitionSvc(&service.AgentDefinitionSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
		PlanGate:        planGate,
		OutboxNotifier:  enqueuer,
	})

	// Chat-run consumer: notification-service signals an agent participant's trigger fired.
	chatRunConsumer := event.NewChatRunConsumer(rabbitmq, inboxRepo, agentDefSvc)
	if err := chatRunConsumer.Listen(ctx); err != nil {
		return err
	}

	// gRPC server
	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	agentgrpc.NewAgentHandler(server.Server(), repoFactory, agentDefSvc, planGate)

	logger.Info("Agent service starting", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
