package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/event"
	"github.com/open-mrp/api/services/core-service/internal/hubspotsync"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/grpc"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/hubspot"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/repository"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/shippo"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	stripeinfra "github.com/open-mrp/api/services/core-service/internal/infrastructure/stripe"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/stub"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/vercel"
	"github.com/open-mrp/api/services/core-service/internal/mediator"
	"github.com/open-mrp/api/services/core-service/internal/service"
	"github.com/open-mrp/api/services/core-service/internal/stripesync"
	s3client "github.com/open-mrp/api/shared/cloud/s3"
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
	slog.SetDefault(logger)

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

	outboxEnqueuerRepo := repository.NewOutboxEnqueuerRepo(db, queries)
	enqueuer, err := messaging.NewEnqueuer(&messaging.EnqueuerConfig{ServiceName: domain.ServiceName, PlatformMode: cfg.PlatformMode}, outboxEnqueuerRepo, rabbitmq, leaseSvc)
	if err != nil {
		return err
	}
	if err := enqueuer.Start(ctx); err != nil {
		return err
	}
	defer enqueuer.Stop()

	// Test mode normally stubs object storage out, but an upload that never stores anything
	// cannot tell you whether the read path looks where the write path put it. When the e2e
	// stack points at a local S3-compatible server, use the real client so those round trips
	// are actually exercised.
	var s3Store s3client.ObjectStore
	if cfg.PlatformMode.IsTest() && !s3client.HasEndpointOverride() {
		s3Store = &s3client.StubClient{}
	} else {
		s3, apiErr := s3client.NewClient(ctx, cfg.AWSRegion)
		if apiErr != nil {
			return fmt.Errorf("failed to create S3 client: %s", apiErr.PublicMessage)
		}
		s3Store = s3
	}

	repoFactory := repository.NewRepoFactory(queries)
	txManager := service.NewTransactionManager(db, queries)

	// Connect to auth-service lazily (no WaitForReady) to avoid a startup deadlock:
	// auth-service waits for core-service to be ready, so core-service cannot
	// block on auth-service here. The gRPC client dials in the background and
	// reconnects on demand; the first tenancy call will establish the connection.
	authClient, err := grpc.NewCoreAuthClient(cfg.AuthServiceURL)
	if err != nil {
		return err
	}
	defer authClient.Close()

	mediatorFactory := mediator.NewMediatorFactory()

	// Branding logos are stored as object keys in the same bucket the upload endpoint writes to, so every document that prints one signs or reads it from here.
	brandingAssets := service.NewBrandingAssets(s3Store, cfg.AccountPhotosBucket)

	// Which accounts carry a marketing sentence in their customer-email footer is operator configuration; install it before anything can render an email.
	blurbs, err := cfg.marketingBlurbs()
	if err != nil {
		return err
	}
	service.SetAccountMarketingBlurbs(blurbs)

	jobSvcFactory := service.NewJobSvcFactory()
	jobSvc := jobSvcFactory.Build(repoFactory)

	accountSvc := service.NewAccountSvc(&service.AccountSvcConfig{
		RepoFactory:         repoFactory,
		MediatorFactory:     mediatorFactory,
		TxManager:           txManager,
		S3Client:            s3Store,
		AccountPhotosBucket: cfg.AccountPhotosBucket,
	})
	sandboxSvc := service.NewSandboxSvc(&service.SandboxSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	unitSvc := service.NewUnitSvc(&service.UnitSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})
	unitGroupSvc := service.NewUnitGroupSvc(&service.UnitGroupSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})
	paymentTermSvc := service.NewPaymentTermSvc(&service.PaymentTermSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	shippingTermSvc := service.NewShippingTermSvc(&service.ShippingTermSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	accountStatusSvc := service.NewAccountStatusSvc(&service.AccountStatusSvcConfig{
		Repos: repoFactory,
	})
	accountGroupSvc := service.NewAccountGroupSvc(&service.AccountGroupSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	accountGroupProductLineAccessSvc := service.NewAccountGroupProductLineAccessSvc(&service.AccountGroupProductLineAccessSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	customerProductLineAccessSvc := service.NewCustomerProductLineAccessSvc(&service.CustomerProductLineAccessSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	productSvc := service.NewProductSvc(&service.ProductSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})
	addressSvc := service.NewAddressSvc(&service.AddressSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	var addressValidationSvc domain.AddressValidationSvc
	if cfg.PlatformMode.IsTest() {
		addressValidationSvc = &stub.AddressValidationSvc{}
	} else {
		addressValidationSvc = service.NewAddressValidationSvc(&service.AddressValidationSvcConfig{
			GoogleMapsAPIKey: cfg.GoogleMapsAPIKey,
		})
	}

	userSvc := service.NewUserSvc(&service.UserSvcConfig{
		Repos:            repoFactory,
		MediatorFactory:  mediatorFactory,
		TxManager:        txManager,
		S3Client:         s3Store,
		UserPhotosBucket: cfg.UserPhotosBucket,
	})

	notificationPublisher := event.NewOutboxNotificationPublisher()
	billingPublisher := event.NewOutboxBillingPublisher()
	accountUserSvc := service.NewAccountUserSvc(&service.AccountUserSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		TxManager:             txManager,
		NotificationPublisher: notificationPublisher,
		BillingPublisher:      billingPublisher,
		S3Client:              s3Store,
		UserPhotosBucket:      cfg.UserPhotosBucket,
		Branding:              brandingAssets,
		PlatformMode:          cfg.PlatformMode,
	})
	accountPriceSvc := service.NewAccountPriceSvc(&service.AccountPriceSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
		Branding:        brandingAssets,
	})
	salesTargetSvc := service.NewSalesTargetSvc(&service.SalesTargetSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	adjustmentTypeSvc := service.NewAdjustmentTypeSvc(&service.AdjustmentTypeSvcConfig{
		Repos: repoFactory,
	})

	prioritySvc := service.NewPrioritySvc(&service.PrioritySvcConfig{
		Repos: repoFactory,
	})

	propertySvc := service.NewPropertySvc(&service.PropertySvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})
	attributeSvc := service.NewAttributeSvc(&service.AttributeSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	var shippoFactory domain.ShippoClientFactory
	var stripeCheckoutFactory domain.StripeCheckoutClientFactory
	var hubspotFactory domain.HubspotClientFactory
	if cfg.PlatformMode.IsTest() {
		shippoFactory = &stub.ShippoClientFactory{}
		stripeCheckoutFactory = &stub.StripeCheckoutClientFactory{}
		hubspotFactory = &stub.HubspotClientFactory{}
	} else {
		shippoFactory = shippo.NewClientFactory()
		stripeCheckoutFactory = stripeinfra.NewCheckoutClientFactory()
		hubspotFactory = hubspot.NewClientFactory()
	}

	// The stub also backs non-production runs without Vercel credentials so make dev works out of the box; config validation guarantees the credentials in production.
	var portalDomainProvider domain.PortalDomainProvider
	if cfg.PlatformMode.IsTest() || cfg.VercelAPIToken == "" {
		portalDomainProvider = stub.NewPortalDomainProvider()
	} else {
		portalDomainProvider = vercel.NewPortalDomainProvider(cfg.VercelAPIToken, cfg.VercelProjectID, cfg.VercelTeamID)
	}
	portalDomainSvc := service.NewPortalDomainSvc(&service.PortalDomainSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
		Provider:        portalDomainProvider,
	})

	var integrationEncryptionKey []byte
	if cfg.IntegrationEncryptionKey != "" {
		var err error
		integrationEncryptionKey, err = hex.DecodeString(cfg.IntegrationEncryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decode INTEGRATION_ENCRYPTION_KEY: %w", err)
		}
	}
	accountIntegrationSvc := service.NewAccountIntegrationSvc(&service.AccountIntegrationSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
		EncryptionKey:   integrationEncryptionKey,
		EncryptionKeyID: cfg.IntegrationEncryptionKeyID,
		PlatformMode:    cfg.PlatformMode,
	})

	hubspotSync := hubspotsync.NewService(repoFactory, hubspotFactory, integrationEncryptionKey, hubspotsync.Config{})
	stripeSync := stripesync.NewService(repoFactory, stripeCheckoutFactory, integrationEncryptionKey)
	hubspotSyncSvc := service.NewHubspotSyncSvc(&service.HubspotSyncSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
		Publisher:       event.NewOutboxHubspotSyncPublisher(),
	})

	carrierSvc := service.NewCarrierSvc(&service.CarrierSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
		ShippoFactory:   shippoFactory,
		EncryptionKey:   integrationEncryptionKey,
		AccountSvc:      accountSvc,
	})
	serviceLevelSvc := service.NewServiceLevelSvc(&service.ServiceLevelSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})
	shippingCaseSvc := service.NewShippingCaseSvc(&service.ShippingCaseSvcConfig{
		RepoFactory:          repoFactory,
		MediatorFactory:      mediatorFactory,
		TxManager:            txManager,
		S3Client:             s3Store,
		ShippingLabelsBucket: cfg.ShippingLabelsBucket,
	})

	itemSvc := service.NewItemSvc(&service.ItemSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	partSvc := service.NewPartSvc(&service.PartSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	childAccountSvc := service.NewChildAccountSvc(&service.ChildAccountSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	batchSvc := service.NewBatchSvc(&service.BatchSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	itemCategorySvc := service.NewItemCategorySvc(&service.ItemCategorySvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	productLineSvc := service.NewProductLineSvc(&service.ProductLineSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	productTypeSvc := service.NewProductTypeSvc(&service.ProductTypeSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	consumptionSvc := service.NewConsumptionSvc(&service.ConsumptionSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	productionFlowSvc := service.NewProductionFlowSvc(&service.ProductionFlowSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	productionStepSvc := service.NewProductionStepSvc(&service.ProductionStepSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	productionSvc := service.NewProductionSvc(&service.ProductionSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	customerSvc := service.NewCustomerSvc(&service.CustomerSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	analyticsSvc := service.NewAnalyticsSvc(&service.AnalyticsSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
	})

	catalogSvc := service.NewCatalogSvc(&service.CatalogSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
	})

	ediSvc := service.NewEDISvc(&service.EDISvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	machineSvc := service.NewMachineSvc(&service.MachineSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	machineStatusSvc := service.NewMachineStatusSvc(&service.MachineStatusSvcConfig{Repos: repoFactory})

	departmentSvc := service.NewDepartmentSvc(&service.DepartmentSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	deliverySvc := service.NewDeliverySvc(&service.DeliverySvcConfig{
		Repos: repoFactory,
	})

	emailLogSvc := service.NewEmailLogSvc(&service.EmailLogSvcConfig{
		Repos: repoFactory,
	})

	inventoryChangeLogSvc := service.NewInventoryChangeLogSvc(&service.InventoryChangeLogSvcConfig{
		Repos: repoFactory,
	})

	invoiceSvc := service.NewInvoiceSvc(&service.InvoiceSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	salesOrderStatusSvc := service.NewSalesOrderStatusSvc(&service.SalesOrderStatusSvcConfig{
		Repos: repoFactory,
	})

	orderDiscountSvc := service.NewOrderDiscountSvc(&service.OrderDiscountSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	volumeDiscountSvc := service.NewVolumeDiscountSvc(&service.VolumeDiscountSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	salesOrderSvc := service.NewSalesOrderSvc(&service.SalesOrderSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		TxManager:             txManager,
		CheckoutClientFactory: stripeCheckoutFactory,
		NotificationPublisher: notificationPublisher,
		SalesOrderPublisher:   event.NewOutboxSalesOrderEventPublisher(),
		ShippoFactory:         shippoFactory,
		EncryptionKey:         integrationEncryptionKey,
		FrontendURL:           cfg.FrontendURL,
		Branding:              brandingAssets,
	})

	salesOrderLineSvc := service.NewSalesOrderLineSvc(&service.SalesOrderLineSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	receivableSvc := service.NewReceivableSvc(&service.ReceivableSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		TxManager:             txManager,
		NotificationPublisher: notificationPublisher,
	})

	settlementSvc := service.NewSettlementSvc(&service.SettlementSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	transactionAllocationSvc := service.NewTransactionAllocationSvc(&service.TransactionAllocationSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	transactionSvc := service.NewTransactionSvc(&service.TransactionSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	purchaseOrderSvc := service.NewPurchaseOrderSvc(&service.PurchaseOrderSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		TxManager:             txManager,
		NotificationPublisher: notificationPublisher,
	})

	purchaseOrderLineSvc := service.NewPurchaseOrderLineSvc(&service.PurchaseOrderLineSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	materialSvc := service.NewMaterialSvc(&service.MaterialSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	permissionGroupSvc := service.NewPermissionGroupSvc(&service.PermissionGroupSvcConfig{
		Repos: repoFactory,
	})

	supplierMaterialSvc := service.NewSupplierMaterialSvc(&service.SupplierMaterialSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	pickSvc := service.NewPickSvc(&service.PickSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	pickLineSvc := service.NewPickLineSvc(&service.PickLineSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	receivingOrderSvc := service.NewReceivingOrderSvc(&service.ReceivingOrderSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	receivingOrderLineSvc := service.NewReceivingOrderLineSvc(&service.ReceivingOrderLineSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	productionRunSvc := service.NewProductionRunSvc(&service.ProductionRunSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	productionScheduleSvc := service.NewProductionScheduleSvc(&service.ProductionScheduleSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
		Enqueuer:        event.NewOutboxProductionScheduleEnqueuer(),
	})

	operatingCalendarSvc := service.NewOperatingCalendarSvc(&service.OperatingCalendarSvcConfig{
		Repos: repoFactory,
	})

	machineDowntimeSvc := service.NewMachineDowntimeSvc(&service.MachineDowntimeSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	demandOverrideSvc := service.NewDemandOverrideSvc(&service.DemandOverrideSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	measureSvc := service.NewMeasureSvc(&service.MeasureSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	roleSvc := service.NewRoleSvc(&service.RoleSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	utilsSvc := service.NewUtilsSvc(&service.UtilsSvcConfig{
		Repos:                 repoFactory,
		MediatorFactory:       mediatorFactory,
		TxManager:             txManager,
		NotificationPublisher: notificationPublisher,
		FrontendURL:           cfg.FrontendURL,
		Branding:              brandingAssets,
	})

	locationSvc := service.NewLocationSvc(&service.LocationSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	scanningStationSvc := service.NewScanningStationSvc(&service.ScanningStationSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       txManager,
	})

	supplierSvc := service.NewSupplierSvc(&service.SupplierSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	sysPropertySvc := service.NewSysPropertySvc(&service.SysPropertySvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	registrationFlowSvc := service.NewRegistrationFlowSvc(&service.RegistrationFlowSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	portalRegistrationSessionSvc := service.NewPortalRegistrationSessionSvc(&service.PortalRegistrationSessionSvcConfig{
		Repos:     repoFactory,
		Registrar: registrationFlowSvc,
	})

	territorySvc := service.NewTerritorySvc(&service.TerritorySvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
	})

	shipmentSvc := service.NewShipmentSvc(&service.ShipmentSvcConfig{
		Repos:                repoFactory,
		MediatorFactory:      mediatorFactory,
		TxManager:            txManager,
		ShippoFactory:        shippoFactory,
		EncryptionKey:        integrationEncryptionKey,
		NotificationPub:      notificationPublisher,
		BillingPub:           billingPublisher,
		S3Client:             s3Store,
		ShippingLabelsBucket: cfg.ShippingLabelsBucket,
		FrontendURL:          cfg.FrontendURL,
		Branding:             brandingAssets,
	})

	shipmentLineSvc := service.NewShipmentLineSvc(&service.ShipmentLineSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       txManager,
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

	execStepConsumer := event.NewExecuteProductionStepConsumer(rabbitmq, inboxRepo, queries, repoFactory, txManager)
	if err := execStepConsumer.Listen(ctx); err != nil {
		return err
	}

	batchScannedConsumer := event.NewBatchScannedConsumer(rabbitmq, inboxRepo, repoFactory, txManager)
	if err := batchScannedConsumer.Listen(ctx); err != nil {
		return err
	}

	inventoryReceivedConsumer := event.NewInventoryReceivedConsumer(rabbitmq, inboxRepo, repoFactory, txManager)
	if err := inventoryReceivedConsumer.Listen(ctx); err != nil {
		return err
	}

	costBasisConsumer := event.NewItemCostBasisChangedConsumer(rabbitmq, inboxRepo, repoFactory, itemSvc)
	if err := costBasisConsumer.Listen(ctx); err != nil {
		return err
	}

	undoBatchScanConsumer := event.NewUndoBatchScanConsumer(rabbitmq, inboxRepo, repoFactory)
	if err := undoBatchScanConsumer.Listen(ctx); err != nil {
		return err
	}

	recalcBurnRateConsumer := event.NewRecalcItemBurnRateConsumer(rabbitmq, inboxRepo, repoFactory, txManager)
	if err := recalcBurnRateConsumer.Listen(ctx); err != nil {
		return err
	}

	allocateOpenIssuesConsumer := event.NewAllocateOpenIssuesConsumer(rabbitmq, inboxRepo, repoFactory, txManager)
	if err := allocateOpenIssuesConsumer.Listen(ctx); err != nil {
		return err
	}

	// Every async bulk operation runs on the same generic consumer; the variance is data —
	// the operation's canonical identity plus its executor. Add a row to register one.
	bulkConsumers := []*event.BulkOperationConsumer{
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkCreateProductionRuns, productionRunSvc.ExecuteBulkCreateProductionRuns),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertProductionSteps, productionStepSvc.ExecuteBulkUpsertProductionSteps),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertUnits, unitSvc.ExecuteBulkUpsertUnits),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertUnitGroups, unitGroupSvc.ExecuteBulkUpsertUnitGroups),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertLocations, locationSvc.ExecuteBulkUpsertLocations),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertDepartments, departmentSvc.ExecuteBulkUpsertDepartments),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertMachines, machineSvc.ExecuteBulkUpsertMachines),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertProductLines, productLineSvc.ExecuteBulkUpsertProductLines),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertScanningStations, scanningStationSvc.ExecuteBulkUpsertScanningStations),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertItemCategories, itemCategorySvc.ExecuteBulkUpsertItemCategories),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertParts, partSvc.ExecuteBulkUpsertParts),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertProducts, productSvc.ExecuteBulkUpsertProducts),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertMaterials, materialSvc.ExecuteBulkUpsertMaterials),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkUpsertProperties, propertySvc.ExecuteBulkUpsertProperties),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.BulkResolveHubspotCompanyReviews, hubspotSyncSvc.ExecuteBulkResolveReviews),
		event.NewBulkOperationConsumer(rabbitmq, inboxRepo, messaging.PackPick, pickSvc.ExecutePackPick),
	}
	for _, bulkConsumer := range bulkConsumers {
		if err := bulkConsumer.Listen(ctx); err != nil {
			return err
		}
	}

	// The reader signs links into the same bucket the runner uploads to, so both sides of
	// an export agree on where its file lives.
	exportDelivery := service.NewExportDelivery(s3Store, cfg.ExportsBucket)
	exportSvc := service.NewExportSvc(&service.ExportSvcConfig{Delivery: exportDelivery})

	// Each builder is a method on the service owning that resource.
	exportBuilders := map[string]domain.ExportBuilder{
		"units":                   unitSvc.BuildExportUnits,
		"unit_groups":             unitGroupSvc.BuildExportUnitGroups,
		"product_lines":           productLineSvc.BuildExportProductLines,
		"item_categories":         itemCategorySvc.BuildExportItemCategories,
		"departments":             departmentSvc.BuildExportDepartments,
		"locations":               locationSvc.BuildExportLocations,
		"machines":                machineSvc.BuildExportMachines,
		"scanning_stations":       scanningStationSvc.BuildExportScanningStations,
		"production_runs":         productionRunSvc.BuildExportProductionRuns,
		"production_steps":        productionStepSvc.BuildExportProductionSteps,
		"parts":                   partSvc.BuildExportParts,
		"products":                productSvc.BuildExportProducts,
		"materials":               materialSvc.BuildExportMaterials,
		"properties":              propertySvc.BuildExportProperties,
		"hubspot_company_reviews": hubspotSyncSvc.BuildExportHubspotCompanyReviews,
		"price_list":              accountPriceSvc.BuildExportPriceList,
	}

	exportRunner := service.NewExportRunner(&service.ExportRunnerConfig{
		Repos:         repoFactory,
		JobSvcFactory: jobSvcFactory,
		Delivery:      exportDelivery,
		Builders:      exportBuilders,
	})
	exportConsumers := make([]*event.ExportConsumer, 0, len(messaging.ExportOperations))
	for _, op := range messaging.ExportOperations {
		exportConsumers = append(exportConsumers, event.NewExportConsumer(rabbitmq, inboxRepo, op, exportRunner.Render))
	}
	for _, exportConsumer := range exportConsumers {
		if err := exportConsumer.Listen(ctx); err != nil {
			return err
		}
	}

	salesOrderCreatedConsumer := event.NewSalesOrderCreatedConsumer(rabbitmq, inboxRepo, hubspotSync, salesOrderSvc)
	if err := salesOrderCreatedConsumer.Listen(ctx); err != nil {
		return err
	}

	// Drains the queue the generation cadence publishes to. Registered unconditionally: a declared queue that nothing consumes accumulates messages forever.
	generateScheduleConsumer := event.NewGenerateProductionScheduleConsumer(rabbitmq, inboxRepo, productionScheduleSvc, repoFactory)
	if err := generateScheduleConsumer.Listen(ctx); err != nil {
		return err
	}

	scheduleCadence := service.NewProductionScheduleScheduler(&service.ProductionScheduleSchedulerConfig{
		Repos: repoFactory,
		Lease: leaseSvc,
		Svc:   productionScheduleSvc,
	})
	if err := scheduleCadence.Start(ctx); err != nil {
		return err
	}
	defer scheduleCadence.Stop()

	salesOrderShippingUpdatedConsumer := event.NewSalesOrderShippingUpdatedConsumer(rabbitmq, inboxRepo, repoFactory, salesOrderSvc)
	if err := salesOrderShippingUpdatedConsumer.Listen(ctx); err != nil {
		return err
	}

	hubspotSyncConsumer := event.NewHubspotSyncConsumer(rabbitmq, inboxRepo, hubspotSync)
	if err := hubspotSyncConsumer.Listen(ctx); err != nil {
		return err
	}

	syncStripeCustomerConsumer := event.NewSyncStripeCustomerConsumer(rabbitmq, inboxRepo, stripeSync)
	if err := syncStripeCustomerConsumer.Listen(ctx); err != nil {
		return err
	}

	server, err := contracts.NewGRPCServer(domain.ServiceName, nil, nil)
	if err != nil {
		return err
	}
	srv := server.Server()
	grpc.RegisterAddressService(srv, addressSvc, addressValidationSvc)
	grpc.RegisterCarrierService(srv, carrierSvc, serviceLevelSvc)
	grpc.RegisterShippingCaseService(srv, shippingCaseSvc)
	grpc.RegisterPortalDomainService(srv, portalDomainSvc)
	grpc.RegisterMiscService(srv, accountIntegrationSvc, adjustmentTypeSvc, emailLogSvc, inventoryChangeLogSvc, prioritySvc)
	grpc.RegisterAccountUserService(srv, accountUserSvc)
	grpc.RegisterAnalyticsService(srv, analyticsSvc)
	grpc.RegisterCatalogService(srv, catalogSvc)
	grpc.RegisterEDIService(srv, ediSvc)
	grpc.RegisterRoleService(srv, roleSvc)
	grpc.RegisterCustomerService(srv, customerSvc, childAccountSvc, customerProductLineAccessSvc, productSvc)
	grpc.RegisterGroupService(srv, accountGroupSvc, accountGroupProductLineAccessSvc)
	grpc.RegisterSalesService(srv, accountPriceSvc, salesTargetSvc, productSvc, invoiceSvc, salesOrderStatusSvc, orderDiscountSvc, volumeDiscountSvc, salesOrderSvc, salesOrderLineSvc, receivableSvc, settlementSvc, transactionAllocationSvc, transactionSvc)
	grpc.RegisterPurchaseService(srv, purchaseOrderSvc, purchaseOrderLineSvc)
	grpc.RegisterFulfillmentService(srv, batchSvc, consumptionSvc, deliverySvc, departmentSvc, machineSvc, machineStatusSvc, productionFlowSvc)
	grpc.RegisterItemService(srv, unitSvc, unitGroupSvc, itemSvc, itemCategorySvc, propertySvc, attributeSvc, paymentTermSvc, shippingTermSvc, partSvc, productLineSvc, productTypeSvc)
	grpc.RegisterMaterialService(srv, materialSvc, supplierMaterialSvc)
	grpc.RegisterPermissionGroupService(srv, permissionGroupSvc)
	grpc.RegisterPickingService(srv, pickSvc, pickLineSvc)
	grpc.RegisterReceivingService(srv, receivingOrderSvc, receivingOrderLineSvc)
	grpc.RegisterProductionRunService(srv, productionRunSvc)
	grpc.RegisterJobService(srv, jobSvc, exportSvc)
	grpc.RegisterProductionStepService(srv, productionStepSvc, productionSvc)
	grpc.RegisterMachineDowntimeService(srv, machineDowntimeSvc)
	grpc.RegisterDemandOverrideService(srv, demandOverrideSvc)
	grpc.RegisterProductionScheduleService(srv, productionScheduleSvc, operatingCalendarSvc)
	grpc.RegisterHubspotSyncService(srv, hubspotSyncSvc)
	grpc.RegisterMeasureService(srv, measureSvc)
	grpc.RegisterUtilsService(srv, utilsSvc)
	grpc.RegisterUserService(srv, userSvc)
	grpc.RegisterLocationService(srv, locationSvc)
	grpc.RegisterScanningStationService(srv, scanningStationSvc)
	grpc.RegisterSupplierService(srv, supplierSvc)
	grpc.RegisterSysPropertyService(srv, sysPropertySvc)
	grpc.RegisterRegistrationFlowService(srv, registrationFlowSvc)
	grpc.RegisterPortalRegistrationSessionService(srv, portalRegistrationSessionSvc)
	grpc.RegisterTerritoryService(srv, territorySvc)
	grpc.RegisterShippingService(srv, shipmentSvc, shipmentLineSvc)
	grpc.RegisterAccountService(srv, accountSvc, sandboxSvc, accountStatusSvc)

	tenancySvc := service.NewTenancySvc(&service.TenancySvcConfig{
		RepoFactory:      repoFactory,
		AuthClient:       authClient,
		S3Client:         s3Store,
		UserPhotosBucket: cfg.UserPhotosBucket,
	})
	grpc.RegisterTenancyService(srv, tenancySvc)

	logger.Info("Core service started", "port", cfg.Port)

	return server.Serve(ctx, cfg.Port)
}
