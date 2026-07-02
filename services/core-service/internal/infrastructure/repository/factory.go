package repository

import (
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/messaging"
)

type repoFactoryImpl struct {
	queries *sqlc.Queries
}

func NewRepoFactory(queries *sqlc.Queries) domain.RepoFactory {
	return &repoFactoryImpl{queries: queries}
}

func (r *repoFactoryImpl) NewAccountRepo() domain.AccountRepo {
	return NewAccountRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountUserRepo() domain.AccountUserRepo {
	return NewAccountUserRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountRelationRepo() domain.AccountRelationRepo {
	return NewAccountRelationRepo(r.queries)
}

func (r *repoFactoryImpl) NewRolePermissionRepo() domain.RolePermissionRepo {
	return NewRolePermissionRepo(r.queries)
}

func (r *repoFactoryImpl) NewRoleRepo() domain.RoleRepo {
	return NewRoleRepo(r.queries)
}

func (r *repoFactoryImpl) NewSandboxAccountRepo() domain.SandboxAccountRepo {
	return NewSandboxAccountRepo(r.queries)
}

func (r *repoFactoryImpl) NewRegistrationRepo() domain.RegistrationRepo {
	return NewRegistrationRepo(r.queries)
}

func (r *repoFactoryImpl) NewIdempotencyKeyRepo() domain.IdempotencyKeyRepo {
	return NewIdempotencyKeyRepo(r.queries)
}

func (r *repoFactoryImpl) NewUnitRepo() domain.UnitRepo {
	return NewUnitRepo(r.queries)
}

func (r *repoFactoryImpl) NewPaymentTermRepo() domain.PaymentTermRepo {
	return NewPaymentTermRepo(r.queries)
}

func (r *repoFactoryImpl) NewShippingTermRepo() domain.ShippingTermRepo {
	return NewShippingTermRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductRepo() domain.ProductRepo {
	return NewProductRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountGroupRepo() domain.AccountGroupRepo {
	return NewAccountGroupRepo(r.queries)
}

func (r *repoFactoryImpl) NewDeletedRecordRepo() domain.DeletedRecordRepo {
	return NewDeletedRecordRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountGroupProductLineAccessRepo() domain.AccountGroupProductLineAccessRepo {
	return NewAccountGroupProductLineAccessRepo(r.queries)
}

func (r *repoFactoryImpl) NewCustomerProductLineAccessRepo() domain.CustomerProductLineAccessRepo {
	return NewCustomerProductLineAccessRepo(r.queries)
}

func (r *repoFactoryImpl) NewAddressRepo() domain.AddressRepo {
	return NewAddressRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountStatusRepo() domain.AccountStatusRepo {
	return NewAccountStatusRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountPriceRepo() domain.AccountPriceRepo {
	return NewAccountPriceRepo(r.queries)
}

func (r *repoFactoryImpl) NewAccountIntegrationRepo() domain.AccountIntegrationRepo {
	return NewAccountIntegrationRepo(r.queries)
}

func (r *repoFactoryImpl) NewHubspotSyncRepo() domain.HubspotSyncRepo {
	return NewHubspotSyncRepo(r.queries)
}

func (r *repoFactoryImpl) NewSalesTargetRepo() domain.SalesTargetRepo {
	return NewSalesTargetRepo(r.queries)
}

func (r *repoFactoryImpl) NewUserRepo() domain.UserRepo {
	return NewUserRepo(r.queries)
}

func (r *repoFactoryImpl) NewAdjustmentTypeRepo() domain.AdjustmentTypeRepo {
	return NewAdjustmentTypeRepo(r.queries)
}

func (r *repoFactoryImpl) NewPropertyRepo() domain.PropertyRepo {
	return NewPropertyRepo(r.queries)
}

func (r *repoFactoryImpl) NewAttributeRepo() domain.AttributeRepo {
	return NewAttributeRepo(r.queries)
}

func (r *repoFactoryImpl) NewCarrierRepo() domain.CarrierRepo {
	return NewCarrierRepo(r.queries)
}

func (r *repoFactoryImpl) NewServiceLevelRepo() domain.ServiceLevelRepo {
	return NewServiceLevelRepo(r.queries)
}

func (r *repoFactoryImpl) NewBatchRepo() domain.BatchRepo {
	return NewBatchRepo(r.queries)
}

func (r *repoFactoryImpl) NewItemRepo() domain.ItemRepo {
	return NewItemRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductionStepQueryRepo() domain.ProductionStepQueryRepo {
	return NewProductionStepQueryRepo(r.queries)
}

func (r *repoFactoryImpl) NewScanningStationQueryRepo() domain.ScanningStationQueryRepo {
	return NewScanningStationQueryRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductionRunQueryRepo() domain.ProductionRunQueryRepo {
	return NewProductionRunQueryRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductionRunRepo() domain.ProductionRunRepo {
	return NewProductionRunRepo(r.queries)
}

func (r *repoFactoryImpl) NewUnitGroupQueryRepo() domain.UnitGroupQueryRepo {
	return NewUnitGroupQueryRepo(r.queries)
}

func (r *repoFactoryImpl) NewUnitGroupRepo() domain.UnitGroupRepo {
	return NewUnitGroupRepo(r.queries)
}

func (r *repoFactoryImpl) NewUnitQueryRepo() domain.UnitQueryRepo {
	return NewUnitQueryRepo(r.queries)
}

func (r *repoFactoryImpl) NewInventoryQueryRepo() domain.InventoryQueryRepo {
	return NewInventoryQueryRepo(r.queries)
}

func (r *repoFactoryImpl) NewItemCategoryRepo() domain.ItemCategoryRepo {
	return NewItemCategoryRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductLineRepo() domain.ProductLineRepo {
	return NewProductLineRepo(r.queries)
}

func (r *repoFactoryImpl) NewConsumptionRepo() domain.ConsumptionRepo {
	return NewConsumptionRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductionFlowRepo() domain.ProductionFlowRepo {
	return NewProductionFlowRepo(r.queries)
}

func (r *repoFactoryImpl) NewCustomerRepo() domain.CustomerRepo {
	return NewCustomerRepo(r.queries)
}

func (r *repoFactoryImpl) NewInventoryMutationRepo() domain.InventoryMutationRepo {
	return NewInventoryMutationRepo(r.queries)
}

func (r *repoFactoryImpl) NewOrderQueryRepo() domain.OrderQueryRepo {
	return NewOrderQueryRepo(r.queries)
}

func (r *repoFactoryImpl) NewInventoryReservationRepo() domain.InventoryReservationRepo {
	return NewInventoryReservationRepo(r.queries)
}

func (r *repoFactoryImpl) NewMaterialDemandRepo() domain.MaterialDemandRepo {
	return NewMaterialDemandRepo(r.queries)
}

func (r *repoFactoryImpl) NewUnitConversionRepo() domain.UnitConversionRepo {
	return NewUnitConversionRepo(r.queries)
}

func (r *repoFactoryImpl) NewMachineRepo() domain.MachineRepo {
	return NewMachineRepo(r.queries)
}

func (r *repoFactoryImpl) NewDepartmentRepo() domain.DepartmentRepo {
	return NewDepartmentRepo(r.queries)
}

func (r *repoFactoryImpl) NewOutboxRepo() messaging.OutboxRepo {
	return NewOutboxRepo(r.queries)
}

func (r *repoFactoryImpl) NewDeliveryRepo() domain.DeliveryRepo {
	return NewDeliveryRepo(r.queries)
}

func (r *repoFactoryImpl) NewEmailLogRepo() domain.EmailLogRepo {
	return NewEmailLogRepo(r.queries)
}

func (r *repoFactoryImpl) NewInventoryChangeLogRepo() domain.InventoryChangeLogRepo {
	return NewInventoryChangeLogRepo(r.queries)
}

func (r *repoFactoryImpl) NewInvoiceRepo() domain.InvoiceRepo {
	return NewInvoiceRepo(r.queries)
}

func (r *repoFactoryImpl) NewReceivableRepo() domain.ReceivableRepo {
	return NewReceivableRepo(r.queries)
}

func (r *repoFactoryImpl) NewSalesOrderRepo() domain.SalesOrderRepo {
	return NewSalesOrderRepo(r.queries)
}

func (r *repoFactoryImpl) NewSalesOrderLineRepo() domain.SalesOrderLineRepo {
	return NewSalesOrderLineRepo(r.queries)
}

func (r *repoFactoryImpl) NewPurchaseOrderRepo() domain.PurchaseOrderRepo {
	return NewPurchaseOrderRepo(r.queries)
}

func (r *repoFactoryImpl) NewPurchaseOrderLineRepo() domain.PurchaseOrderLineRepo {
	return NewPurchaseOrderLineRepo(r.queries)
}

func (r *repoFactoryImpl) NewReceivingOrderRepo() domain.ReceivingOrderRepo {
	return NewReceivingOrderRepo(r.queries)
}

func (r *repoFactoryImpl) NewSalesOrderStatusRepo() domain.SalesOrderStatusRepo {
	return NewSalesOrderStatusRepo(r.queries)
}

func (r *repoFactoryImpl) NewOrderDiscountRepo() domain.OrderDiscountRepo {
	return NewOrderDiscountRepo(r.queries)
}

func (r *repoFactoryImpl) NewVolumeDiscountRepo() domain.VolumeDiscountRepo {
	return NewVolumeDiscountRepo(r.queries)
}

func (r *repoFactoryImpl) NewMaterialRepo() domain.MaterialRepo {
	return NewMaterialRepo(r.queries)
}

func (r *repoFactoryImpl) NewSupplierMaterialRepo() domain.SupplierMaterialRepo {
	return NewSupplierMaterialRepo(r.queries)
}

func (r *repoFactoryImpl) NewPartRepo() domain.PartRepo {
	return NewPartRepo(r.queries)
}

func (r *repoFactoryImpl) NewPermissionGroupRepo() domain.PermissionGroupRepo {
	return NewPermissionGroupRepo(r.queries)
}

func (r *repoFactoryImpl) NewPickRepo() domain.PickRepo {
	return NewPickRepo(r.queries)
}

func (r *repoFactoryImpl) NewPickLineRepo() domain.PickLineRepo {
	return NewPickLineRepo(r.queries)
}

func (r *repoFactoryImpl) NewPriorityRepo() domain.PriorityRepo {
	return NewPriorityRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductTypeRepo() domain.ProductTypeRepo {
	return NewProductTypeRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductionStepRepo() domain.ProductionStepRepo {
	return NewProductionStepRepo(r.queries)
}

func (r *repoFactoryImpl) NewProductionRepo() domain.ProductionRepo {
	return NewProductionRepo(r.queries)
}

func (r *repoFactoryImpl) NewQuantityRepo() domain.QuantityRepo {
	return NewQuantityRepo(r.queries)
}

func (r *repoFactoryImpl) NewRateRepo() domain.RateRepo {
	return NewRateRepo(r.queries)
}

func (r *repoFactoryImpl) NewSettlementRepo() domain.SettlementRepo {
	return NewSettlementRepo(r.queries)
}

func (r *repoFactoryImpl) NewTransactionRepo() domain.TransactionRepo {
	return NewTransactionRepo(r.queries)
}

func (r *repoFactoryImpl) NewTransactionAllocationRepo() domain.TransactionAllocationRepo {
	return NewTransactionAllocationRepo(r.queries)
}

func (r *repoFactoryImpl) NewAnalyticsRepo() domain.AnalyticsRepo {
	return NewAnalyticsRepo(r.queries)
}

func (r *repoFactoryImpl) NewCatalogRepo() domain.CatalogRepo {
	return NewCatalogRepo(r.queries)
}

func (r *repoFactoryImpl) NewEDIRepo() domain.EDIRepo {
	return NewEDIRepo(r.queries)
}

func (r *repoFactoryImpl) NewRegistrationFlowRepo() domain.RegistrationFlowRepo {
	return NewRegistrationFlowRepo(r.queries)
}

func (r *repoFactoryImpl) NewShippingCaseRepo() domain.ShippingCaseRepo {
	return NewShippingCaseRepo(r.queries)
}

func (r *repoFactoryImpl) NewSysPropertyRepo() domain.SysPropertyRepo {
	return NewSysPropertyRepo(r.queries)
}

func (r *repoFactoryImpl) NewStripeEventLogRepo() domain.StripeEventLogRepo {
	return NewStripeEventLogRepo(r.queries)
}

func (r *repoFactoryImpl) NewOrderPaymentIntentRepo() domain.OrderPaymentIntentRepo {
	return NewOrderPaymentIntentRepo(r.queries)
}

func (r *repoFactoryImpl) NewCustomerRegistrationRepo() domain.CustomerRegistrationRepo {
	return NewCustomerRegistrationRepo(r.queries)
}

func (r *repoFactoryImpl) NewShipmentRepo() domain.ShipmentRepo {
	return NewShipmentRepo(r.queries)
}

func (r *repoFactoryImpl) NewShipmentLineRepo() domain.ShipmentLineRepo {
	return NewShipmentLineRepo(r.queries)
}

func (r *repoFactoryImpl) NewTerritoryRepo() domain.TerritoryRepo {
	return NewTerritoryRepo(r.queries)
}

func (r *repoFactoryImpl) NewSupplierRepo() domain.SupplierRepo {
	return NewSupplierRepo(r.queries)
}

func (r *repoFactoryImpl) NewLocationRepo() domain.LocationRepo {
	return NewLocationRepo(r.queries)
}

func (r *repoFactoryImpl) NewScanningStationRepo() domain.ScanningStationRepo {
	return NewScanningStationRepo(r.queries)
}

func (r *repoFactoryImpl) NewPricingRepo() domain.PricingRepo {
	return NewPricingRepo(r.queries)
}
