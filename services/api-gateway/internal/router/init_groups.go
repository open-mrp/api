package router

import (
	"log"
	"net/http"
	"time"

	"github.com/augno/api/services/api-gateway/internal/middleware"
	httpgroup "github.com/augno/api/services/api-gateway/pkg/group"
)

func (r *router) InitEndpointGroups(config MainRouterConfig) {
	registry := NewRegistry()

	// Main endpoints backed by the Auth gRPC service.
	if config.AuthClient == nil {
		panic("Main router: Auth client is a nil pointer")
	}

	// Setup middleware
	middlewareLogger := log.New(config.LogWriter, config.LogPrefix, config.LogFlags)
	requestLogSaver := middleware.NewRequestLogSaver(config.RequestLogPublisher)
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middlewareLogger, next, requestLogSaver, r, config.TrustedProxyHops)
	}
	authMiddlewareConfig := &middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	}
	idempotencyMiddlewareConfig := &middleware.IdempotencyMiddlewareConfig{
		PlatformClient: config.PlatformClient,
	}

	// Middlewares
	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.IPBlockMiddleware(config.TrustedProxyHops))
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(middleware.ExternalHostMiddleware())
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.CORSMiddleware())
	r.AddMiddleware(middleware.RateLimitMiddleware(config.TrustedProxyHops))
	if config.Internal {
		// Internal listener: trust the agent identity supplied by agent-service (gated by the shared service token) instead of validating a user credential against auth-service.
		r.AddMiddleware(middleware.InternalAuthMiddleware(&middleware.InternalAuthMiddlewareConfig{
			ServiceToken: config.InternalServiceToken,
		}))
	} else {
		r.AddMiddleware(middleware.AuthMiddleware(authMiddlewareConfig))
	}
	r.AddMiddleware(middleware.SubscriptionMiddleware())
	r.AddMiddleware(middleware.SandboxBillingMiddleware())
	r.AddMiddleware(middleware.VersionMiddleware())
	r.AddMiddleware(middleware.IdempotencyMiddleware(idempotencyMiddlewareConfig))
	r.AddMiddleware(middleware.RecoverMiddleware())

	// Healthz
	healthGroup := (&httpgroup.HealthEndpointGroup{}).Materialize(httpgroup.HealthEndpointGroupConfig{})
	if healthGroup != nil {
		registry.RegisterGroup(healthGroup.APIEndpointGroup)
	}

	// Sandboxes
	sandboxesGroup := (&httpgroup.SandboxesEndpointGroup{}).Materialize(&httpgroup.SandboxesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if sandboxesGroup != nil {
		registry.RegisterGroup(sandboxesGroup.APIEndpointGroup)
	}

	// Billing
	billingGroup := (&httpgroup.BillingEndpointGroup{}).Materialize(&httpgroup.BillingEndpointGroupConfig{
		BillingClient: config.BillingClient,
		CoreClient:    config.CoreClient,
	})
	if billingGroup != nil {
		registry.RegisterGroup(billingGroup.APIEndpointGroup)
	}

	// Units
	unitsGroup := (&httpgroup.UnitsEndpointGroup{}).Materialize(&httpgroup.UnitsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if unitsGroup != nil {
		registry.RegisterGroup(unitsGroup.APIEndpointGroup)
	}

	// Unit Groups
	unitGroupsGroup := (&httpgroup.UnitGroupsEndpointGroup{}).Materialize(&httpgroup.UnitGroupsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if unitGroupsGroup != nil {
		registry.RegisterGroup(unitGroupsGroup.APIEndpointGroup)
	}

	// Payment Terms
	paymentTermsGroup := (&httpgroup.PaymentTermsEndpointGroup{}).Materialize(&httpgroup.PaymentTermsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if paymentTermsGroup != nil {
		registry.RegisterGroup(paymentTermsGroup.APIEndpointGroup)
	}

	// Shipping Terms
	shippingTermsGroup := (&httpgroup.ShippingTermsEndpointGroup{}).Materialize(&httpgroup.ShippingTermsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if shippingTermsGroup != nil {
		registry.RegisterGroup(shippingTermsGroup.APIEndpointGroup)
	}

	// Addresses
	addressesGroup := (&httpgroup.AddressesEndpointGroup{}).Materialize(&httpgroup.AddressesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if addressesGroup != nil {
		registry.RegisterGroup(addressesGroup.APIEndpointGroup)
	}

	// Address Validation
	addressValidationGroup := (&httpgroup.AddressValidationEndpointGroup{}).Materialize(&httpgroup.AddressValidationEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if addressValidationGroup != nil {
		registry.RegisterGroup(addressValidationGroup.APIEndpointGroup)
	}

	// Accounts
	accountsGroup := (&httpgroup.AccountsEndpointGroup{}).Materialize(&httpgroup.AccountsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if accountsGroup != nil {
		registry.RegisterGroup(accountsGroup.APIEndpointGroup)
	}

	// Portal Domains (custom domains for customer portals)
	portalDomainsGroup := (&httpgroup.PortalDomainsEndpointGroup{}).Materialize(&httpgroup.PortalDomainsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if portalDomainsGroup != nil {
		registry.RegisterGroup(portalDomainsGroup.APIEndpointGroup)
	}

	// Portal Registration Sessions
	portalRegistrationSessionsGroup := (&httpgroup.PortalRegistrationSessionsEndpointGroup{}).Materialize(&httpgroup.PortalRegistrationSessionsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if portalRegistrationSessionsGroup != nil {
		registry.RegisterGroup(portalRegistrationSessionsGroup.APIEndpointGroup)
	}

	// Account Statuses
	accountStatusesGroup := (&httpgroup.AccountStatusesEndpointGroup{}).Materialize(&httpgroup.AccountStatusesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if accountStatusesGroup != nil {
		registry.RegisterGroup(accountStatusesGroup.APIEndpointGroup)
	}

	// Account Users
	accountUsersGroup := (&httpgroup.AccountUsersEndpointGroup{}).Materialize(&httpgroup.AccountUsersEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if accountUsersGroup != nil {
		registry.RegisterGroup(accountUsersGroup.APIEndpointGroup)
	}

	// Sales Targets
	salesTargetsGroup := (&httpgroup.SalesTargetsEndpointGroup{}).Materialize(&httpgroup.SalesTargetsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if salesTargetsGroup != nil {
		registry.RegisterGroup(salesTargetsGroup.APIEndpointGroup)
	}

	// Users
	usersGroup := (&httpgroup.UsersEndpointGroup{}).Materialize(&httpgroup.UsersEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if usersGroup != nil {
		registry.RegisterGroup(usersGroup.APIEndpointGroup)
	}

	// Account Groups
	accountGroupsGroup := (&httpgroup.AccountGroupsEndpointGroup{}).Materialize(&httpgroup.AccountGroupsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if accountGroupsGroup != nil {
		registry.RegisterGroup(accountGroupsGroup.APIEndpointGroup)
	}

	// Account Group Product Line Access
	accountGroupProductLineAccessGroup := (&httpgroup.AccountGroupProductLineAccessEndpointGroup{}).Materialize(&httpgroup.AccountGroupProductLineAccessEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if accountGroupProductLineAccessGroup != nil {
		registry.RegisterGroup(accountGroupProductLineAccessGroup.APIEndpointGroup)
	}

	// Customer Product Line Access
	customerProductLineAccessGroup := (&httpgroup.CustomerProductLineAccessEndpointGroup{}).Materialize(&httpgroup.CustomerProductLineAccessEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if customerProductLineAccessGroup != nil {
		registry.RegisterGroup(customerProductLineAccessGroup.APIEndpointGroup)
	}

	// Priorities
	prioritiesGroup := (&httpgroup.PrioritiesEndpointGroup{}).Materialize(&httpgroup.PrioritiesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if prioritiesGroup != nil {
		registry.RegisterGroup(prioritiesGroup.APIEndpointGroup)
	}

	// Account Integrations
	accountIntegrationsGroup := (&httpgroup.AccountIntegrationsEndpointGroup{}).Materialize(&httpgroup.AccountIntegrationsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if accountIntegrationsGroup != nil {
		registry.RegisterGroup(accountIntegrationsGroup.APIEndpointGroup)
	}

	// HubSpot Sync (backfill/reconciliation)
	hubspotSyncGroup := (&httpgroup.HubspotSyncEndpointGroup{}).Materialize(&httpgroup.HubspotSyncEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if hubspotSyncGroup != nil {
		registry.RegisterGroup(hubspotSyncGroup.APIEndpointGroup)
	}

	// Carriers
	carriersGroup := (&httpgroup.CarriersEndpointGroup{}).Materialize(&httpgroup.CarriersEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if carriersGroup != nil {
		registry.RegisterGroup(carriersGroup.APIEndpointGroup)
	}

	// Service Levels
	serviceLevelsGroup := (&httpgroup.ServiceLevelsEndpointGroup{}).Materialize(&httpgroup.ServiceLevelsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if serviceLevelsGroup != nil {
		registry.RegisterGroup(serviceLevelsGroup.APIEndpointGroup)
	}

	// Account Prices
	accountPricesGroup := (&httpgroup.AccountPricesEndpointGroup{}).Materialize(&httpgroup.AccountPricesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if accountPricesGroup != nil {
		registry.RegisterGroup(accountPricesGroup.APIEndpointGroup)
	}

	// Items
	itemsGroup := (&httpgroup.ItemsEndpointGroup{}).Materialize(&httpgroup.ItemsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if itemsGroup != nil {
		registry.RegisterGroup(itemsGroup.APIEndpointGroup)
	}

	// Inventories
	inventoriesGroup := (&httpgroup.InventoriesEndpointGroup{}).Materialize(&httpgroup.InventoriesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if inventoriesGroup != nil {
		registry.RegisterGroup(inventoriesGroup.APIEndpointGroup)
	}

	// Materials
	materialsGroup := (&httpgroup.MaterialsEndpointGroup{}).Materialize(&httpgroup.MaterialsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if materialsGroup != nil {
		registry.RegisterGroup(materialsGroup.APIEndpointGroup)
	}

	// Supplier Materials
	supplierMaterialsGroup := (&httpgroup.SupplierMaterialsEndpointGroup{}).Materialize(&httpgroup.SupplierMaterialsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if supplierMaterialsGroup != nil {
		registry.RegisterGroup(supplierMaterialsGroup.APIEndpointGroup)
	}

	// Parts
	partsGroup := (&httpgroup.PartsEndpointGroup{}).Materialize(&httpgroup.PartsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if partsGroup != nil {
		registry.RegisterGroup(partsGroup.APIEndpointGroup)
	}

	// Child Accounts
	childAccountsGroup := (&httpgroup.ChildAccountsEndpointGroup{}).Materialize(&httpgroup.ChildAccountsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if childAccountsGroup != nil {
		registry.RegisterGroup(childAccountsGroup.APIEndpointGroup)
	}

	// Customers
	customersGroup := (&httpgroup.CustomersEndpointGroup{}).Materialize(&httpgroup.CustomersEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if customersGroup != nil {
		registry.RegisterGroup(customersGroup.APIEndpointGroup)
	}

	// Contacts
	contactsGroup := (&httpgroup.ContactsEndpointGroup{}).Materialize(&httpgroup.ContactsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if contactsGroup != nil {
		registry.RegisterGroup(contactsGroup.APIEndpointGroup)
	}

	// Deliveries
	deliveriesGroup := (&httpgroup.DeliveriesEndpointGroup{}).Materialize(&httpgroup.DeliveriesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if deliveriesGroup != nil {
		registry.RegisterGroup(deliveriesGroup.APIEndpointGroup)
	}

	// Product Lines
	productLinesGroup := (&httpgroup.ProductLinesEndpointGroup{}).Materialize(&httpgroup.ProductLinesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if productLinesGroup != nil {
		registry.RegisterGroup(productLinesGroup.APIEndpointGroup)
	}

	// Products
	productsGroup := (&httpgroup.ProductsEndpointGroup{}).Materialize(&httpgroup.ProductsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if productsGroup != nil {
		registry.RegisterGroup(productsGroup.APIEndpointGroup)
	}

	// Item Categories
	itemCategoriesGroup := (&httpgroup.ItemCategoriesEndpointGroup{}).Materialize(&httpgroup.ItemCategoriesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if itemCategoriesGroup != nil {
		registry.RegisterGroup(itemCategoriesGroup.APIEndpointGroup)
	}

	// Consumptions
	consumptionsGroup := (&httpgroup.ConsumptionsEndpointGroup{}).Materialize(&httpgroup.ConsumptionsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if consumptionsGroup != nil {
		registry.RegisterGroup(consumptionsGroup.APIEndpointGroup)
	}

	// Production Steps
	productionStepsGroup := (&httpgroup.ProductionStepsEndpointGroup{}).Materialize(&httpgroup.ProductionStepsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if productionStepsGroup != nil {
		registry.RegisterGroup(productionStepsGroup.APIEndpointGroup)
	}

	// Production Flows
	productionFlowsGroup := (&httpgroup.ProductionFlowsEndpointGroup{}).Materialize(&httpgroup.ProductionFlowsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if productionFlowsGroup != nil {
		registry.RegisterGroup(productionFlowsGroup.APIEndpointGroup)
	}

	// Batches
	batchesGroup := (&httpgroup.BatchesEndpointGroup{}).Materialize(&httpgroup.BatchesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if batchesGroup != nil {
		registry.RegisterGroup(batchesGroup.APIEndpointGroup)
	}

	// Departments
	departmentsGroup := (&httpgroup.DepartmentsEndpointGroup{}).Materialize(&httpgroup.DepartmentsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if departmentsGroup != nil {
		registry.RegisterGroup(departmentsGroup.APIEndpointGroup)
	}

	// Email Logs
	emailLogsGroup := (&httpgroup.EmailLogsEndpointGroup{}).Materialize(&httpgroup.EmailLogsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if emailLogsGroup != nil {
		registry.RegisterGroup(emailLogsGroup.APIEndpointGroup)
	}

	// Inventory Change Logs
	inventoryChangeLogsGroup := (&httpgroup.InventoryChangeLogsEndpointGroup{}).Materialize(&httpgroup.InventoryChangeLogsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if inventoryChangeLogsGroup != nil {
		registry.RegisterGroup(inventoryChangeLogsGroup.APIEndpointGroup)
	}

	// Invoices
	invoicesGroup := (&httpgroup.InvoicesEndpointGroup{}).Materialize(&httpgroup.InvoicesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if invoicesGroup != nil {
		registry.RegisterGroup(invoicesGroup.APIEndpointGroup)
	}

	// Receivables
	receivablesGroup := (&httpgroup.ReceivablesEndpointGroup{}).Materialize(&httpgroup.ReceivablesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if receivablesGroup != nil {
		registry.RegisterGroup(receivablesGroup.APIEndpointGroup)
	}

	// Machines
	machinesGroup := (&httpgroup.MachinesEndpointGroup{}).Materialize(&httpgroup.MachinesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if machinesGroup != nil {
		registry.RegisterGroup(machinesGroup.APIEndpointGroup)
	}

	// Order Discounts
	orderDiscountsGroup := (&httpgroup.OrderDiscountsEndpointGroup{}).Materialize(&httpgroup.OrderDiscountsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if orderDiscountsGroup != nil {
		registry.RegisterGroup(orderDiscountsGroup.APIEndpointGroup)
	}

	// Transactions
	transactionsGroup := (&httpgroup.TransactionsEndpointGroup{}).Materialize(&httpgroup.TransactionsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if transactionsGroup != nil {
		registry.RegisterGroup(transactionsGroup.APIEndpointGroup)
	}

	// Production Runs
	productionRunsGroup := (&httpgroup.ProductionRunsEndpointGroup{}).Materialize(&httpgroup.ProductionRunsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if productionRunsGroup != nil {
		registry.RegisterGroup(productionRunsGroup.APIEndpointGroup)
	}

	// Machine Downtime
	machineDowntimeGroup := (&httpgroup.MachineDowntimeEndpointGroup{}).Materialize(&httpgroup.MachineDowntimeEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if machineDowntimeGroup != nil {
		registry.RegisterGroup(machineDowntimeGroup.APIEndpointGroup)
	}

	// Machine Status
	machineStatusGroup := (&httpgroup.MachineStatusEndpointGroup{}).Materialize(&httpgroup.MachineStatusEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if machineStatusGroup != nil {
		registry.RegisterGroup(machineStatusGroup.APIEndpointGroup)
	}

	// Production Schedules
	productionSchedulesGroup := (&httpgroup.ProductionSchedulesEndpointGroup{}).Materialize(&httpgroup.ProductionSchedulesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if productionSchedulesGroup != nil {
		registry.RegisterGroup(productionSchedulesGroup.APIEndpointGroup)
	}

	// Production Schedule Settings
	scheduleSettingsGroup := (&httpgroup.ProductionScheduleSettingsEndpointGroup{}).Materialize(&httpgroup.ProductionScheduleSettingsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if scheduleSettingsGroup != nil {
		registry.RegisterGroup(scheduleSettingsGroup.APIEndpointGroup)
	}

	// Demand Overrides
	demandOverridesGroup := (&httpgroup.DemandOverridesEndpointGroup{}).Materialize(&httpgroup.DemandOverridesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if demandOverridesGroup != nil {
		registry.RegisterGroup(demandOverridesGroup.APIEndpointGroup)
	}

	// Volume Discounts
	volumeDiscountsGroup := (&httpgroup.VolumeDiscountsEndpointGroup{}).Materialize(&httpgroup.VolumeDiscountsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if volumeDiscountsGroup != nil {
		registry.RegisterGroup(volumeDiscountsGroup.APIEndpointGroup)
	}

	// Sales Orders
	salesOrdersGroup := (&httpgroup.SalesOrdersEndpointGroup{}).Materialize(&httpgroup.SalesOrdersEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if salesOrdersGroup != nil {
		registry.RegisterGroup(salesOrdersGroup.APIEndpointGroup)
	}

	// Purchase Orders
	purchaseOrdersGroup := (&httpgroup.PurchaseOrdersEndpointGroup{}).Materialize(&httpgroup.PurchaseOrdersEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if purchaseOrdersGroup != nil {
		registry.RegisterGroup(purchaseOrdersGroup.APIEndpointGroup)
	}

	// Picks
	picksGroup := (&httpgroup.PicksEndpointGroup{}).Materialize(&httpgroup.PicksEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if picksGroup != nil {
		registry.RegisterGroup(picksGroup.APIEndpointGroup)
	}

	// Receiving Orders
	receivingOrdersGroup := (&httpgroup.ReceivingOrdersEndpointGroup{}).Materialize(&httpgroup.ReceivingOrdersEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if receivingOrdersGroup != nil {
		registry.RegisterGroup(receivingOrdersGroup.APIEndpointGroup)
	}

	// Properties
	propertiesGroup := (&httpgroup.PropertiesEndpointGroup{}).Materialize(&httpgroup.PropertiesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if propertiesGroup != nil {
		registry.RegisterGroup(propertiesGroup.APIEndpointGroup)
	}

	// Product Types
	productTypesGroup := (&httpgroup.ProductTypesEndpointGroup{}).Materialize(&httpgroup.ProductTypesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if productTypesGroup != nil {
		registry.RegisterGroup(productTypesGroup.APIEndpointGroup)
	}

	// Sales Order Statuses
	salesOrderStatusesGroup := (&httpgroup.SalesOrderStatusesEndpointGroup{}).Materialize(&httpgroup.SalesOrderStatusesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if salesOrderStatusesGroup != nil {
		registry.RegisterGroup(salesOrderStatusesGroup.APIEndpointGroup)
	}

	// Permission Groups
	permissionGroupsGroup := (&httpgroup.PermissionGroupsEndpointGroup{}).Materialize(&httpgroup.PermissionGroupsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if permissionGroupsGroup != nil {
		registry.RegisterGroup(permissionGroupsGroup.APIEndpointGroup)
	}

	// Request Logs
	requestLogsGroup := (&httpgroup.RequestLogsEndpointGroup{}).Materialize(&httpgroup.RequestLogsEndpointGroupConfig{
		PlatformClient: config.PlatformClient,
	})
	if requestLogsGroup != nil {
		registry.RegisterGroup(requestLogsGroup.APIEndpointGroup)
	}

	// Audit Events
	auditEventsGroup := (&httpgroup.AuditEventsEndpointGroup{}).Materialize(&httpgroup.AuditEventsEndpointGroupConfig{
		PlatformClient: config.PlatformClient,
	})
	if auditEventsGroup != nil {
		registry.RegisterGroup(auditEventsGroup.APIEndpointGroup)
	}

	// Quantities
	quantitiesGroup := (&httpgroup.QuantitiesEndpointGroup{}).Materialize(&httpgroup.QuantitiesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if quantitiesGroup != nil {
		registry.RegisterGroup(quantitiesGroup.APIEndpointGroup)
	}

	// Rates
	ratesGroup := (&httpgroup.RatesEndpointGroup{}).Materialize(&httpgroup.RatesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if ratesGroup != nil {
		registry.RegisterGroup(ratesGroup.APIEndpointGroup)
	}

	// Tenancy
	tenancyGroup := (&httpgroup.TenancyEndpointGroup{}).Materialize(&httpgroup.TenancyEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if tenancyGroup != nil {
		registry.RegisterGroup(tenancyGroup.APIEndpointGroup)
	}

	// System Properties
	sysPropertiesGroup := (&httpgroup.SysPropertiesEndpointGroup{}).Materialize(&httpgroup.SysPropertiesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if sysPropertiesGroup != nil {
		registry.RegisterGroup(sysPropertiesGroup.APIEndpointGroup)
	}

	// Checkout Sessions
	checkoutSessionsGroup := (&httpgroup.CheckoutSessionsEndpointGroup{}).Materialize(&httpgroup.CheckoutSessionsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if checkoutSessionsGroup != nil {
		registry.RegisterGroup(checkoutSessionsGroup.APIEndpointGroup)
	}

	// Registration Flows
	registrationFlowsGroup := (&httpgroup.RegistrationFlowsEndpointGroup{}).Materialize(&httpgroup.RegistrationFlowsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if registrationFlowsGroup != nil {
		registry.RegisterGroup(registrationFlowsGroup.APIEndpointGroup)
	}

	// Roles
	rolesGroup := (&httpgroup.RolesEndpointGroup{}).Materialize(&httpgroup.RolesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if rolesGroup != nil {
		registry.RegisterGroup(rolesGroup.APIEndpointGroup)
	}

	// Territories
	territoriesGroup := (&httpgroup.TerritoriesEndpointGroup{}).Materialize(&httpgroup.TerritoriesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if territoriesGroup != nil {
		registry.RegisterGroup(territoriesGroup.APIEndpointGroup)
	}

	// Suppliers
	suppliersGroup := (&httpgroup.SuppliersEndpointGroup{}).Materialize(&httpgroup.SuppliersEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if suppliersGroup != nil {
		registry.RegisterGroup(suppliersGroup.APIEndpointGroup)
	}

	// Locations
	locationsGroup := (&httpgroup.LocationsEndpointGroup{}).Materialize(&httpgroup.LocationsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if locationsGroup != nil {
		registry.RegisterGroup(locationsGroup.APIEndpointGroup)
	}

	// Scanning Stations
	scanningStationsGroup := (&httpgroup.ScanningStationsEndpointGroup{}).Materialize(&httpgroup.ScanningStationsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if scanningStationsGroup != nil {
		registry.RegisterGroup(scanningStationsGroup.APIEndpointGroup)
	}

	// Shipping Cases
	shippingCasesGroup := (&httpgroup.ShippingCasesEndpointGroup{}).Materialize(&httpgroup.ShippingCasesEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if shippingCasesGroup != nil {
		registry.RegisterGroup(shippingCasesGroup.APIEndpointGroup)
	}

	// Shipments
	shipmentsGroup := (&httpgroup.ShipmentsEndpointGroup{}).Materialize(&httpgroup.ShipmentsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if shipmentsGroup != nil {
		registry.RegisterGroup(shipmentsGroup.APIEndpointGroup)
	}

	// Transaction Allocations
	transactionAllocationsGroup := (&httpgroup.TransactionAllocationsEndpointGroup{}).Materialize(&httpgroup.TransactionAllocationsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if transactionAllocationsGroup != nil {
		registry.RegisterGroup(transactionAllocationsGroup.APIEndpointGroup)
	}

	// Settlements
	settlementsGroup := (&httpgroup.SettlementsEndpointGroup{}).Materialize(&httpgroup.SettlementsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if settlementsGroup != nil {
		registry.RegisterGroup(settlementsGroup.APIEndpointGroup)
	}

	// Notifications (in-app bell feed)
	if config.NotificationClient != nil {
		notificationsGroup := (&httpgroup.NotificationsEndpointGroup{}).Materialize(&httpgroup.NotificationsEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if notificationsGroup != nil {
			registry.RegisterGroup(notificationsGroup.APIEndpointGroup)
		}

		// Announcements (broadcast feed)
		announcementsGroup := (&httpgroup.AnnouncementsEndpointGroup{}).Materialize(&httpgroup.AnnouncementsEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if announcementsGroup != nil {
			registry.RegisterGroup(announcementsGroup.APIEndpointGroup)
		}

		// Conversations (1:1 chat)
		conversationsGroup := (&httpgroup.ConversationsEndpointGroup{}).Materialize(&httpgroup.ConversationsEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if conversationsGroup != nil {
			registry.RegisterGroup(conversationsGroup.APIEndpointGroup)
		}

		// Support Routes (which group conversation handles a relationship's support)
		supportRoutesGroup := (&httpgroup.SupportRoutesEndpointGroup{}).Materialize(&httpgroup.SupportRoutesEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if supportRoutesGroup != nil {
			registry.RegisterGroup(supportRoutesGroup.APIEndpointGroup)
		}

		// Unified resource search (e.g. chat @-mention picker)
		searchGroup := (&httpgroup.SearchEndpointGroup{}).Materialize(&httpgroup.SearchEndpointGroupConfig{
			CoreClient:         config.CoreClient,
			NotificationClient: config.NotificationClient,
			AgentClient:        config.AgentClient,
		})
		if searchGroup != nil {
			registry.RegisterGroup(searchGroup.APIEndpointGroup)
		}

		// Messages (send, list, edit, delete)
		messagesGroup := (&httpgroup.MessagesEndpointGroup{}).Materialize(&httpgroup.MessagesEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if messagesGroup != nil {
			registry.RegisterGroup(messagesGroup.APIEndpointGroup)
		}

		// Conversation participants (members and agents)
		conversationParticipantsGroup := (&httpgroup.ConversationParticipantsEndpointGroup{}).Materialize(&httpgroup.ConversationParticipantsEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if conversationParticipantsGroup != nil {
			registry.RegisterGroup(conversationParticipantsGroup.APIEndpointGroup)
		}

		// Messaging groups (reusable rosters that seed conversations)
		messagingGroupsGroup := (&httpgroup.MessagingGroupsEndpointGroup{}).Materialize(&httpgroup.MessagingGroupsEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if messagingGroupsGroup != nil {
			registry.RegisterGroup(messagingGroupsGroup.APIEndpointGroup)
		}

		// Message attachments (presigned upload targets)
		messageAttachmentsGroup := (&httpgroup.MessageAttachmentsEndpointGroup{}).Materialize(&httpgroup.MessageAttachmentsEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if messageAttachmentsGroup != nil {
			registry.RegisterGroup(messageAttachmentsGroup.APIEndpointGroup)
		}

		// Message blocks (block/unblock direct messaging)
		messageBlocksGroup := (&httpgroup.MessageBlocksEndpointGroup{}).Materialize(&httpgroup.MessageBlocksEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if messageBlocksGroup != nil {
			registry.RegisterGroup(messageBlocksGroup.APIEndpointGroup)
		}

		// Notification preferences (per-category channel toggles)
		notificationPreferencesGroup := (&httpgroup.NotificationPreferencesEndpointGroup{}).Materialize(&httpgroup.NotificationPreferencesEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if notificationPreferencesGroup != nil {
			registry.RegisterGroup(notificationPreferencesGroup.APIEndpointGroup)
		}

		// Messaging contacts (the messaging directory)
		messagingContactsGroup := (&httpgroup.MessagingContactsEndpointGroup{}).Materialize(&httpgroup.MessagingContactsEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if messagingContactsGroup != nil {
			registry.RegisterGroup(messagingContactsGroup.APIEndpointGroup)
		}

		// Email domains (register + verify sending/receiving domains)
		emailDomainsGroup := (&httpgroup.EmailDomainsEndpointGroup{}).Materialize(&httpgroup.EmailDomainsEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if emailDomainsGroup != nil {
			registry.RegisterGroup(emailDomainsGroup.APIEndpointGroup)
		}

		// Email inboxes (routable inboxes bound to chat conversations)
		emailInboxesGroup := (&httpgroup.EmailInboxesEndpointGroup{}).Materialize(&httpgroup.EmailInboxesEndpointGroupConfig{
			NotificationClient: config.NotificationClient,
		})
		if emailInboxesGroup != nil {
			registry.RegisterGroup(emailInboxesGroup.APIEndpointGroup)
		}
	}

	// Agents
	if config.AgentClient != nil {
		agentsGroup := (&httpgroup.AgentsEndpointGroup{}).Materialize(&httpgroup.AgentsEndpointGroupConfig{
			AgentClient: config.AgentClient,
		})
		if agentsGroup != nil {
			registry.RegisterGroup(agentsGroup.APIEndpointGroup)
		}

		agentRunsGroup := (&httpgroup.AgentRunsEndpointGroup{}).Materialize(&httpgroup.AgentRunsEndpointGroupConfig{
			AgentClient: config.AgentClient,
		})
		if agentRunsGroup != nil {
			registry.RegisterGroup(agentRunsGroup.APIEndpointGroup)
		}

		agentToolsGroup := (&httpgroup.AgentToolsEndpointGroup{}).Materialize(&httpgroup.AgentToolsEndpointGroupConfig{
			AgentClient: config.AgentClient,
		})
		if agentToolsGroup != nil {
			registry.RegisterGroup(agentToolsGroup.APIEndpointGroup)
		}

		agentMemoriesGroup := (&httpgroup.AgentMemoriesEndpointGroup{}).Materialize(&httpgroup.AgentMemoriesEndpointGroupConfig{
			AgentClient: config.AgentClient,
		})
		if agentMemoriesGroup != nil {
			registry.RegisterGroup(agentMemoriesGroup.APIEndpointGroup)
		}
	}

	// EDI
	ediGroup := (&httpgroup.EDIEndpointGroup{}).Materialize(&httpgroup.EDIEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if ediGroup != nil {
		registry.RegisterGroup(ediGroup.APIEndpointGroup)
	}

	ediDCLocationsGroup := (&httpgroup.EDIDCLocationsEndpointGroup{}).Materialize(&httpgroup.EDIDCLocationsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if ediDCLocationsGroup != nil {
		registry.RegisterGroup(ediDCLocationsGroup.APIEndpointGroup)
	}

	ediRunsGroup := (&httpgroup.EDIRunsEndpointGroup{}).Materialize(&httpgroup.EDIRunsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if ediRunsGroup != nil {
		registry.RegisterGroup(ediRunsGroup.APIEndpointGroup)
	}

	// Catalog
	catalogGroup := (&httpgroup.CatalogEndpointGroup{}).Materialize(&httpgroup.CatalogEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if catalogGroup != nil {
		registry.RegisterGroup(catalogGroup.APIEndpointGroup)
	}

	// Analytics
	analyticsGroup := (&httpgroup.AnalyticsEndpointGroup{}).Materialize(&httpgroup.AnalyticsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if analyticsGroup != nil {
		registry.RegisterGroup(analyticsGroup.APIEndpointGroup)
	}

	// Utils
	utilsGroup := (&httpgroup.UtilsEndpointGroup{}).Materialize(&httpgroup.UtilsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if utilsGroup != nil {
		registry.RegisterGroup(utilsGroup.APIEndpointGroup)
	}

	// Records (cross-record document generation, e.g. pack lists)
	recordsGroup := (&httpgroup.RecordsEndpointGroup{}).Materialize(&httpgroup.RecordsEndpointGroupConfig{
		CoreClient: config.CoreClient,
	})
	if recordsGroup != nil {
		registry.RegisterGroup(recordsGroup.APIEndpointGroup)
	}

	registry.RegisterEndpoints(r)
}

func (r *router) InitWebhookEndpointGroups(config WebhookRouterConfig) {
	registry := NewRegistry()

	// Setup middleware — minimal chain for webhooks: no auth, no CORS, no idempotency
	middlewareLogger := log.New(config.LogWriter, config.LogPrefix, config.LogFlags)
	requestLogSaver := middleware.NewRequestLogSaver(config.RequestLogPublisher)
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middlewareLogger, next, requestLogSaver, r, config.TrustedProxyHops)
	}

	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.IPBlockMiddleware(config.TrustedProxyHops))
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(middleware.ExternalHostMiddleware())
	r.AddMiddleware(middleware.RateLimitMiddlewareWithConfig(100, time.Second, config.TrustedProxyHops))
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.RecoverMiddleware())

	// Webhooks
	webhooksGroup := (&httpgroup.WebhooksEndpointGroup{}).Materialize(&httpgroup.WebhooksEndpointGroupConfig{
		BillingClient: config.BillingClient,
		CoreClient:    config.CoreClient,
	})
	if webhooksGroup != nil {
		registry.RegisterGroup(webhooksGroup.APIEndpointGroup)
	}

	registry.RegisterEndpoints(r)
}

func (r *router) InitAuthEndpointGroups(config AuthRouterConfig) {
	registry := NewRegistry()

	// Auth endpoints backed by the Auth gRPC service.
	if config.AuthClient == nil {
		panic("Auth router: Auth client is a nil pointer")
	}

	middlewareLogger := log.New(config.LogWriter, config.LogPrefix, config.LogFlags)
	requestLogSaver := middleware.NewRequestLogSaver(config.RequestLogPublisher)
	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.LoggingMiddleware(middlewareLogger, next, requestLogSaver, r, config.TrustedProxyHops)
	}
	idempotencyMiddlewareConfig := &middleware.IdempotencyMiddlewareConfig{
		PlatformClient: config.PlatformClient,
	}

	// Middlewares
	r.AddMiddleware(middleware.TracingMiddleware())
	r.AddMiddleware(middleware.IPBlockMiddleware(config.TrustedProxyHops))
	r.AddMiddleware(middleware.PlatformMiddleware(config.PlatformMode))
	r.AddMiddleware(middleware.ExternalHostMiddleware())
	r.AddMiddleware(loggingMiddleware)
	r.AddMiddleware(middleware.CORSMiddleware())
	r.AddMiddleware(middleware.RateLimitMiddleware(config.TrustedProxyHops))
	r.AddMiddleware(middleware.AuthSecurityMiddleware())
	r.AddMiddleware(middleware.VersionMiddleware())
	r.AddMiddleware(middleware.IdempotencyMiddleware(idempotencyMiddlewareConfig))
	r.AddMiddleware(middleware.RecoverMiddleware())

	// Auth
	authGroup := (&httpgroup.AuthEndpointGroup{}).Materialize(&httpgroup.AuthEndpointGroupConfig{
		AuthClient: config.AuthClient,
		CoreClient: config.CoreClient,
	})
	if authGroup != nil {
		registry.RegisterGroup(authGroup.APIEndpointGroup)
	}

	// API Keys
	apiKeysGroup := (&httpgroup.APIKeysEndpointGroup{}).Materialize(&httpgroup.APIKeysEndpointGroupConfig{
		AuthClient: config.AuthClient,
	})
	if apiKeysGroup != nil {
		registry.RegisterGroup(apiKeysGroup.APIEndpointGroup)
	}

	// Registration Sessions
	registrationSessionsGroup := (&httpgroup.RegistrationSessionsEndpointGroup{}).Materialize(&httpgroup.RegistrationSessionsEndpointGroupConfig{
		AuthClient: config.AuthClient,
	})
	if registrationSessionsGroup != nil {
		registry.RegisterGroup(registrationSessionsGroup.APIEndpointGroup)
	}

	registry.RegisterEndpoints(r)
}
