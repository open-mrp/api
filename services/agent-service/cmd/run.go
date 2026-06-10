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
	encCfg := &messaging.EnqueuerConfig{
		ServiceName:  domain.ServiceName,
		PlatformMode: cfg.PlatformMode,
	}
	if !cfg.PlatformMode.IsTest() {
		encCfg.PollInterval = 1 * time.Second
	}
	enqueuer, err := messaging.NewEnqueuer(encCfg, outboxRepo, rabbitmq, leaseSvc)
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
	if cfg.PlatformMode.IsTest() {
		stubProvider := &stub.LLMProvider{}
		providers = map[string]llm.LLMProvider{
			"anthropic": stubProvider,
			"openai":    stubProvider,
		}
	} else {
		gateway := llm.NewGatewayProvider(cfg.StripeSecretKey)
		providers = map[string]llm.LLMProvider{
			"anthropic": gateway,
			"openai":    gateway,
		}
	}

	// Repo factory
	repoFactory := repository.NewRepoFactory(queries)

	// Tool handler registry
	toolRegistry := agents.NewToolHandlerRegistry()
	agents.RegisterTools(toolRegistry)

	// Runner service
	runner := service.NewRunnerSvc(&service.RunnerConfig{
		Repos:         repoFactory,
		ToolRegistry:  toolRegistry,
		LLMProviders:  providers,
		OutboxRepo:    repoFactory.NewOutboxRepo(),
		CoreClient:    coreClient,
		Broker:        rabbitmq,
		BillingClient: billingClient,
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
	})

	// gRPC server
	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	agentgrpc.NewAgentHandler(server.Server(), repoFactory, agentDefSvc, planGate)

	logger.Info("Agent service starting", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
