package domain

import "github.com/augno/api/shared/messaging"

type RepoFactory interface {
	NewAccountRepo() AccountRepo
	NewAccountUserRepo() AccountUserRepo
	NewAccountRelationRepo() AccountRelationRepo
	NewRolePermissionRepo() RolePermissionRepo
	NewRoleRepo() RoleRepo
	NewSandboxAccountRepo() SandboxAccountRepo
	NewRegistrationRepo() RegistrationRepo
	NewIdempotencyKeyRepo() IdempotencyKeyRepo
	NewUnitRepo() UnitRepo
	NewPaymentTermRepo() PaymentTermRepo
	NewShippingTermRepo() ShippingTermRepo
	NewProductRepo() ProductRepo
	NewAccountGroupRepo() AccountGroupRepo
	NewDeletedRecordRepo() DeletedRecordRepo
	NewAccountGroupProductLineAccessRepo() AccountGroupProductLineAccessRepo
	NewCustomerProductLineAccessRepo() CustomerProductLineAccessRepo
	NewAddressRepo() AddressRepo
	NewAccountStatusRepo() AccountStatusRepo
	NewUserRepo() UserRepo
	NewAccountPriceRepo() AccountPriceRepo
	NewAccountIntegrationRepo() AccountIntegrationRepo
	NewHubspotSyncRepo() HubspotSyncRepo
	NewSalesTargetRepo() SalesTargetRepo
	NewAdjustmentTypeRepo() AdjustmentTypeRepo
	NewPropertyRepo() PropertyRepo
	NewAttributeRepo() AttributeRepo
	NewCarrierRepo() CarrierRepo
	NewServiceLevelRepo() ServiceLevelRepo
	NewItemRepo() ItemRepo
	NewItemCategoryRepo() ItemCategoryRepo
	NewProductLineRepo() ProductLineRepo
	NewOutboxRepo() messaging.OutboxRepo
	NewBatchRepo() BatchRepo
	NewProductionStepQueryRepo() ProductionStepQueryRepo
	NewScanningStationQueryRepo() ScanningStationQueryRepo
	NewProductionRunQueryRepo() ProductionRunQueryRepo
	NewProductionRunRepo() ProductionRunRepo
	NewUnitGroupQueryRepo() UnitGroupQueryRepo
	NewUnitGroupRepo() UnitGroupRepo
	NewUnitQueryRepo() UnitQueryRepo
	NewInventoryQueryRepo() InventoryQueryRepo
	NewConsumptionRepo() ConsumptionRepo
	NewProductionFlowRepo() ProductionFlowRepo
	NewInventoryMutationRepo() InventoryMutationRepo
	NewOrderQueryRepo() OrderQueryRepo
	NewInventoryReservationRepo() InventoryReservationRepo
	NewMaterialDemandRepo() MaterialDemandRepo
	NewUnitConversionRepo() UnitConversionRepo
	NewCustomerRepo() CustomerRepo
	NewMachineRepo() MachineRepo
	NewDepartmentRepo() DepartmentRepo
	NewDeliveryRepo() DeliveryRepo
	NewEmailLogRepo() EmailLogRepo
	NewInventoryChangeLogRepo() InventoryChangeLogRepo
	NewInvoiceRepo() InvoiceRepo
	NewReceivableRepo() ReceivableRepo
	NewSalesOrderRepo() SalesOrderRepo
	NewSalesOrderLineRepo() SalesOrderLineRepo
	NewSalesOrderStatusRepo() SalesOrderStatusRepo
	NewPurchaseOrderRepo() PurchaseOrderRepo
	NewPurchaseOrderLineRepo() PurchaseOrderLineRepo
	NewReceivingOrderRepo() ReceivingOrderRepo
	NewOrderDiscountRepo() OrderDiscountRepo
	NewVolumeDiscountRepo() VolumeDiscountRepo
	NewMaterialRepo() MaterialRepo
	NewSupplierMaterialRepo() SupplierMaterialRepo
	NewPartRepo() PartRepo
	NewPermissionGroupRepo() PermissionGroupRepo
	NewPickRepo() PickRepo
	NewPickLineRepo() PickLineRepo
	NewPriorityRepo() PriorityRepo
	NewProductTypeRepo() ProductTypeRepo
	NewProductionStepRepo() ProductionStepRepo
	NewProductionRepo() ProductionRepo
	NewQuantityRepo() QuantityRepo
	NewRateRepo() RateRepo
	NewSettlementRepo() SettlementRepo
	NewTransactionRepo() TransactionRepo
	NewTransactionAllocationRepo() TransactionAllocationRepo
	NewAnalyticsRepo() AnalyticsRepo
	NewCatalogRepo() CatalogRepo
	NewEDIRepo() EDIRepo
	NewRegistrationFlowRepo() RegistrationFlowRepo
	NewShippingCaseRepo() ShippingCaseRepo
	NewSysPropertyRepo() SysPropertyRepo
	NewStripeEventLogRepo() StripeEventLogRepo
	NewOrderPaymentIntentRepo() OrderPaymentIntentRepo
	NewCustomerRegistrationRepo() CustomerRegistrationRepo
	NewShipmentRepo() ShipmentRepo
	NewShipmentLineRepo() ShipmentLineRepo
	NewTerritoryRepo() TerritoryRepo
	NewSupplierRepo() SupplierRepo
	NewLocationRepo() LocationRepo
	NewScanningStationRepo() ScanningStationRepo
	NewPricingRepo() PricingRepo
}

// Mediators groups all mediator dependencies built for a specific repository factory.
type Mediators struct {
	Sandbox        SandboxMed
	Idempotency    IdempotencyMed
	ReadAccess     ReadAccessMed
	EditAccess     EditAccessMed
	ProductionFlow ProductionFlowMed
	BurnRate       BurnRateMed
}

// MediatorFactory builds mediators bound to a given repository factory (e.g., per transaction).
type MediatorFactory interface {
	Build(RepoFactory) Mediators
}
