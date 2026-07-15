package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/email"
	"github.com/augno/api/services/notification-service/internal/event"
	"github.com/augno/api/services/notification-service/internal/infrastructure/aws"
	notificationgrpc "github.com/augno/api/services/notification-service/internal/infrastructure/grpc"
	"github.com/augno/api/services/notification-service/internal/infrastructure/repository"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/services/notification-service/internal/infrastructure/stub"
	"github.com/augno/api/services/notification-service/internal/reaper"
	"github.com/augno/api/services/notification-service/internal/scheduler"
	"github.com/augno/api/services/notification-service/internal/service"
	s3client "github.com/augno/api/shared/cloud/s3"
	sqsclient "github.com/augno/api/shared/cloud/sqs"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/lease"
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
	repoFactory := repository.NewRepoFactory(queries)
	txManager := service.NewTransactionManager(db, queries)

	leaseSvc := lease.New(repository.NewLeaseRepo(queries))

	outboxEnqueuerRepo := repository.NewOutboxEnqueuerRepo(db, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{
		ServiceName:  domain.ServiceName,
		PlatformMode: cfg.PlatformMode,
		// Posting a message kicks the enqueuer (see NewConversationSvc) so agent dispatch and realtime
		// delivery fire instantly; this tightens the idle-backoff ceiling below the shared default so an
		// un-kicked realtime event still posts within 500ms.
		MaxPollInterval: 500 * time.Millisecond,
	}, outboxEnqueuerRepo, rabbitmq, leaseSvc)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	templateRenderer, apiErr := email.NewTemplateRenderer()
	if apiErr != nil {
		return apiErr
	}

	notificationConfig, apiErr := service.BuildNotificationSvcConfig(repoFactory, cfg.PlatformMode, cfg.AWSRegion, templateRenderer)
	if apiErr != nil {
		return apiErr
	}
	notificationSvc := service.NewNotificationSvc(notificationConfig)
	messagingSvc := service.NewMessagingSvc(repoFactory)

	var objectStore s3client.ObjectStore
	if cfg.PlatformMode.IsTest() {
		// Chat attachment validation (FileExists) must pass in test mode so the upload→attach flow works.
		objectStore = &s3client.StubClient{FileExistsResult: true}
	} else {
		s3Client, apiErr := s3client.NewClient(ctx, cfg.AWSRegion)
		if apiErr != nil {
			return apiErr
		}
		objectStore = s3Client
	}
	// bridgeEmailSender sends outbound email-bridge replies via SES in the inbound (receiving) region so
	// the reply comes from the same DKIM-verified identity. Nil in test/dev-without-AWS → SendInboxReply
	// errors out there.
	var bridgeEmailSender domain.EmailSender
	if !cfg.PlatformMode.IsTest() {
		bridgeEmailSender, apiErr = aws.NewSESEmailSender(ctx, cfg.PlatformMode, cfg.InboundEmailRegion)
		if apiErr != nil {
			return apiErr
		}
	}
	chatSvc := service.NewConversationSvc(repoFactory, txManager, objectStore, cfg.ChatBucket, rabbitmq, enqueuer, bridgeEmailSender, cfg.InboundEmailDomain)

	var emailIdentityProvider domain.EmailIdentityProvider
	if cfg.PlatformMode.IsTest() {
		emailIdentityProvider = &stub.EmailIdentityProvider{}
	} else {
		// Domain identities are verified in the inbound (receiving) region so the receipt rule accepts
		// their mail; SES email receiving isn't offered in the us-east-2 send region.
		emailIdentityProvider, apiErr = aws.NewSESIdentityProvider(ctx, cfg.InboundEmailRegion)
		if apiErr != nil {
			return apiErr
		}
	}
	emailBridgeSvc := service.NewEmailBridgeSvc(repoFactory, emailIdentityProvider)

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

	messagingReaperRepo := repository.NewMessagingReaperRepo(queries)
	messagingReaper, err := reaper.NewMessagingReaper(&reaper.MessagingReaperConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode}, messagingReaperRepo, leaseSvc, objectStore, cfg.ChatBucket)
	if err != nil {
		return err
	}
	if err := messagingReaper.Start(ctx); err != nil {
		return err
	}
	defer messagingReaper.Stop()

	scheduledMessageWorker, err := scheduler.NewScheduledMessageWorker(&scheduler.ScheduledMessageWorkerConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode}, chatSvc, leaseSvc)
	if err != nil {
		return err
	}
	if err := scheduledMessageWorker.Start(ctx); err != nil {
		return err
	}
	defer scheduledMessageWorker.Stop()

	consumerTracer := workerTracer.Tracer(domain.ServiceName + ".consumer")
	notificationConsumer := event.NewNotificationConsumer(rabbitmq, notificationSvc, inboxRepo, templateRenderer, consumerTracer)
	if err := notificationConsumer.Listen(ctx); err != nil {
		return err
	}

	messagingConsumer := event.NewMessagingConsumer(rabbitmq, messagingSvc, inboxRepo, consumerTracer)
	if err := messagingConsumer.Listen(ctx); err != nil {
		return err
	}

	customerRegisteredConsumer := event.NewCustomerRegisteredConsumer(rabbitmq, messagingSvc, inboxRepo, consumerTracer)
	if err := customerRegisteredConsumer.Listen(ctx); err != nil {
		return err
	}

	agentReplyConsumer := event.NewAgentReplyConsumer(rabbitmq, chatSvc, inboxRepo, consumerTracer)
	if err := agentReplyConsumer.Listen(ctx); err != nil {
		return err
	}

	// Streaming partial-body patches for in-flight agent replies (best-effort, not inbox-deduped).
	agentReplyPatchConsumer := event.NewAgentReplyPatchConsumer(rabbitmq, chatSvc, consumerTracer)
	if err := agentReplyPatchConsumer.Listen(ctx); err != nil {
		return err
	}

	// Inbound email bridge: poll SQS for S3 events, parse the raw mail, thread it into chat. Runs only
	// when a bucket + queue are configured (skipped in local dev without AWS). The inbound bucket/queue
	// live in INBOUND_EMAIL_REGION (us-east-1, the SES receiving region), distinct from AWS_REGION.
	if !cfg.PlatformMode.IsTest() && cfg.InboundEmailBucket != "" && cfg.InboundEmailQueueURL != "" {
		inboundStore, apiErr := s3client.NewClient(ctx, cfg.InboundEmailRegion)
		if apiErr != nil {
			return apiErr
		}
		inboundQueue, apiErr := sqsclient.NewClient(ctx, cfg.InboundEmailRegion, cfg.InboundEmailQueueURL)
		if apiErr != nil {
			return apiErr
		}
		inboundEmailConsumer := event.NewInboundEmailConsumer(inboundQueue, inboundStore, chatSvc, consumerTracer)
		if err := inboundEmailConsumer.Listen(ctx); err != nil {
			return err
		}
		logger.Info("Inbound email consumer started", "queue", cfg.InboundEmailQueueURL, "region", cfg.InboundEmailRegion)
	}

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	notificationgrpc.NewGRPCHandler(server.Server(), notificationSvc)
	notificationgrpc.NewMessagingGRPCHandler(server.Server(), messagingSvc)
	notificationgrpc.NewChatGRPCHandler(server.Server(), chatSvc)
	notificationgrpc.NewEmailBridgeGRPCHandler(server.Server(), emailBridgeSvc, chatSvc, cfg.InboundEmailDomain)

	logger.Info("Notification service started", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
