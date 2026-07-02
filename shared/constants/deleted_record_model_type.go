package constants

// DeletedRecordResourceType represents a resource type stored in deleted_record.
type DeletedRecordResourceType string

const (
	// DeletedRecordResourceTypeAccountGroup identifies deleted account_group records.
	DeletedRecordResourceTypeAccountGroup DeletedRecordResourceType = "account_group"
	// DeletedRecordResourceTypeAccountGroupProductLineAccess identifies deleted account_group_product_line_access records.
	DeletedRecordResourceTypeAccountGroupProductLineAccess DeletedRecordResourceType = "account_group_product_line_access"
	// DeletedRecordResourceTypeAccountIntegration identifies deleted account_integration records.
	DeletedRecordResourceTypeAccountIntegration DeletedRecordResourceType = "account_integration"
	// DeletedRecordResourceTypeAccountPrice identifies deleted account_price records.
	DeletedRecordResourceTypeAccountPrice DeletedRecordResourceType = "account_price"
	// DeletedRecordResourceTypeAccountUser identifies deleted account_user records.
	DeletedRecordResourceTypeAccountUser DeletedRecordResourceType = "account_user"
	// DeletedRecordResourceTypeAddress identifies deleted address records.
	DeletedRecordResourceTypeAddress DeletedRecordResourceType = "address"
	// DeletedRecordResourceTypeAttribute identifies deleted attribute records.
	DeletedRecordResourceTypeAttribute DeletedRecordResourceType = "attribute"
	// DeletedRecordResourceTypeBatch identifies deleted batch records.
	DeletedRecordResourceTypeBatch DeletedRecordResourceType = "batch"
	// DeletedRecordResourceTypeCarrier identifies deleted carrier records.
	DeletedRecordResourceTypeCarrier DeletedRecordResourceType = "carrier"
	// DeletedRecordResourceTypeServiceLevel identifies deleted service_level records.
	DeletedRecordResourceTypeServiceLevel DeletedRecordResourceType = "service_level"
	// DeletedRecordResourceTypeConsumption identifies deleted consumption records.
	DeletedRecordResourceTypeConsumption DeletedRecordResourceType = "consumption"
	// DeletedRecordResourceTypeCustomAgent identifies deleted custom_agent records.
	DeletedRecordResourceTypeCustomAgent DeletedRecordResourceType = "custom_agent"
	// DeletedRecordResourceTypeAgentMemory identifies deleted agent_memory records.
	DeletedRecordResourceTypeAgentMemory DeletedRecordResourceType = "agent_memory"
	// DeletedRecordResourceTypeMessagingGroup identifies deleted messaging_group records.
	DeletedRecordResourceTypeMessagingGroup DeletedRecordResourceType = "messaging_group"
	// DeletedRecordResourceTypeCustomer identifies deleted customer records.
	DeletedRecordResourceTypeCustomer DeletedRecordResourceType = "customer"
	// DeletedRecordResourceTypeCustomerProductLineAccess identifies deleted customer_product_line_access records.
	DeletedRecordResourceTypeCustomerProductLineAccess DeletedRecordResourceType = "customer_product_line_access"
	// DeletedRecordResourceTypeDCLocation identifies deleted dc_location records.
	DeletedRecordResourceTypeDCLocation DeletedRecordResourceType = "dc_location"
	// DeletedRecordResourceTypeDepartment identifies deleted department records.
	DeletedRecordResourceTypeDepartment DeletedRecordResourceType = "department"
	// DeletedRecordResourceTypeItemCategory identifies deleted item_category records.
	DeletedRecordResourceTypeItemCategory DeletedRecordResourceType = "item_category"
	// DeletedRecordResourceTypeMachine identifies deleted machine records.
	DeletedRecordResourceTypeMachine DeletedRecordResourceType = "machine"
	// DeletedRecordResourceTypeMaterial identifies deleted material records.
	DeletedRecordResourceTypeMaterial DeletedRecordResourceType = "material"
	// DeletedRecordResourceTypeOrderDiscount identifies deleted order_discount records.
	DeletedRecordResourceTypeOrderDiscount DeletedRecordResourceType = "order_discount"
	// DeletedRecordResourceTypePart identifies deleted part records.
	DeletedRecordResourceTypePart DeletedRecordResourceType = "part"
	// DeletedRecordResourceTypePaymentTerm identifies deleted payment_term records.
	DeletedRecordResourceTypePaymentTerm DeletedRecordResourceType = "payment_term"
	// DeletedRecordResourceTypeProduct identifies deleted product records.
	DeletedRecordResourceTypeProduct DeletedRecordResourceType = "product"
	// DeletedRecordResourceTypeProductLine identifies deleted product_line records.
	DeletedRecordResourceTypeProductLine DeletedRecordResourceType = "product_line"
	// DeletedRecordResourceTypeProductType identifies deleted product_type records.
	DeletedRecordResourceTypeProductType DeletedRecordResourceType = "product_type"
	// DeletedRecordResourceTypeProductionRun identifies deleted production_run records.
	DeletedRecordResourceTypeProductionRun DeletedRecordResourceType = "production_run"
	// DeletedRecordResourceTypeProductionStep identifies deleted production_step records.
	DeletedRecordResourceTypeProductionStep DeletedRecordResourceType = "production_step"
	// DeletedRecordResourceTypeProperty identifies deleted property records.
	DeletedRecordResourceTypeProperty DeletedRecordResourceType = "property"
	// DeletedRecordResourceTypePurchaseOrder identifies deleted purchase_order records.
	DeletedRecordResourceTypePurchaseOrder DeletedRecordResourceType = "purchase_order"
	// DeletedRecordResourceTypePurchaseOrderLine identifies deleted purchase_order_line records.
	DeletedRecordResourceTypePurchaseOrderLine DeletedRecordResourceType = "purchase_order_line"
	// DeletedRecordResourceTypeRegistrationFlow identifies deleted registration_flow records.
	DeletedRecordResourceTypeRegistrationFlow DeletedRecordResourceType = "registration_flow"
	// DeletedRecordResourceTypeRole identifies deleted role records.
	DeletedRecordResourceTypeRole DeletedRecordResourceType = "role"
	// DeletedRecordResourceTypeSalesOrder identifies deleted sales_order records.
	DeletedRecordResourceTypeSalesOrder DeletedRecordResourceType = "sales_order"
	// DeletedRecordResourceTypeSalesOrderLine identifies deleted sales_order_line records.
	DeletedRecordResourceTypeSalesOrderLine DeletedRecordResourceType = "sales_order_line"
	// DeletedRecordResourceTypeSandbox identifies deleted sandbox records.
	DeletedRecordResourceTypeSandbox DeletedRecordResourceType = "sandbox"
	// DeletedRecordResourceTypeScanningStation identifies deleted scanning_station records.
	DeletedRecordResourceTypeScanningStation DeletedRecordResourceType = "scanning_station"
	// DeletedRecordResourceTypeSettlement identifies deleted settlement records.
	DeletedRecordResourceTypeSettlement DeletedRecordResourceType = "settlement"
	// DeletedRecordResourceTypeShipment identifies deleted shipment records.
	DeletedRecordResourceTypeShipment DeletedRecordResourceType = "shipment"
	// DeletedRecordResourceTypeShipmentLine identifies deleted shipment_line records.
	DeletedRecordResourceTypeShipmentLine DeletedRecordResourceType = "shipment_line"
	// DeletedRecordResourceTypeShippingCase identifies deleted shipping_case records.
	DeletedRecordResourceTypeShippingCase DeletedRecordResourceType = "shipping_case"
	// DeletedRecordResourceTypeShippingTerm identifies deleted shipping_term records.
	DeletedRecordResourceTypeShippingTerm DeletedRecordResourceType = "shipping_term"
	// DeletedRecordResourceTypeLocation identifies deleted storage_location records.
	DeletedRecordResourceTypeLocation DeletedRecordResourceType = "storage_location"
	// DeletedRecordResourceTypeSupplier identifies deleted supplier records.
	DeletedRecordResourceTypeSupplier DeletedRecordResourceType = "supplier"
	// DeletedRecordResourceTypeSupplierMaterial identifies deleted supplier_material records.
	DeletedRecordResourceTypeSupplierMaterial DeletedRecordResourceType = "supplier_material"
	// DeletedRecordResourceTypeTerritory identifies deleted territory records.
	DeletedRecordResourceTypeTerritory DeletedRecordResourceType = "territory"
	// DeletedRecordResourceTypeTransaction identifies deleted transaction records.
	DeletedRecordResourceTypeTransaction DeletedRecordResourceType = "transaction"
	// DeletedRecordResourceTypeTransactionAllocation identifies deleted transaction_allocation records.
	DeletedRecordResourceTypeTransactionAllocation DeletedRecordResourceType = "transaction_allocation"
	// DeletedRecordResourceTypeUnit identifies deleted unit records.
	DeletedRecordResourceTypeUnit DeletedRecordResourceType = "unit"
	// DeletedRecordResourceTypeUnitGroup identifies deleted unit_group records.
	DeletedRecordResourceTypeUnitGroup DeletedRecordResourceType = "unit_group"
	// DeletedRecordResourceTypeUnitGroupUnit identifies deleted unit_group_unit records.
	DeletedRecordResourceTypeUnitGroupUnit DeletedRecordResourceType = "unit_group_unit"
	// DeletedRecordResourceTypeVolumeDiscount identifies deleted volume_discount records.
	DeletedRecordResourceTypeVolumeDiscount DeletedRecordResourceType = "volume_discount"
)

func (m DeletedRecordResourceType) IsValid() bool {
	switch m {
	case DeletedRecordResourceTypeAccountGroup,
		DeletedRecordResourceTypeAccountGroupProductLineAccess,
		DeletedRecordResourceTypeAccountIntegration,
		DeletedRecordResourceTypeAccountPrice,
		DeletedRecordResourceTypeAccountUser,
		DeletedRecordResourceTypeAddress,
		DeletedRecordResourceTypeAttribute,
		DeletedRecordResourceTypeBatch,
		DeletedRecordResourceTypeCarrier,
		DeletedRecordResourceTypeServiceLevel,
		DeletedRecordResourceTypeConsumption,
		DeletedRecordResourceTypeCustomAgent,
		DeletedRecordResourceTypeAgentMemory,
		DeletedRecordResourceTypeMessagingGroup,
		DeletedRecordResourceTypeCustomer,
		DeletedRecordResourceTypeCustomerProductLineAccess,
		DeletedRecordResourceTypeDCLocation,
		DeletedRecordResourceTypeDepartment,
		DeletedRecordResourceTypeItemCategory,
		DeletedRecordResourceTypeMachine,
		DeletedRecordResourceTypeMaterial,
		DeletedRecordResourceTypeOrderDiscount,
		DeletedRecordResourceTypePart,
		DeletedRecordResourceTypePaymentTerm,
		DeletedRecordResourceTypeProduct,
		DeletedRecordResourceTypeProductLine,
		DeletedRecordResourceTypeProductType,
		DeletedRecordResourceTypeProductionRun,
		DeletedRecordResourceTypeProductionStep,
		DeletedRecordResourceTypeProperty,
		DeletedRecordResourceTypePurchaseOrder,
		DeletedRecordResourceTypePurchaseOrderLine,
		DeletedRecordResourceTypeRegistrationFlow,
		DeletedRecordResourceTypeRole,
		DeletedRecordResourceTypeSalesOrder,
		DeletedRecordResourceTypeSalesOrderLine,
		DeletedRecordResourceTypeSandbox,
		DeletedRecordResourceTypeScanningStation,
		DeletedRecordResourceTypeSettlement,
		DeletedRecordResourceTypeShipment,
		DeletedRecordResourceTypeShipmentLine,
		DeletedRecordResourceTypeShippingCase,
		DeletedRecordResourceTypeShippingTerm,
		DeletedRecordResourceTypeLocation,
		DeletedRecordResourceTypeSupplier,
		DeletedRecordResourceTypeSupplierMaterial,
		DeletedRecordResourceTypeTerritory,
		DeletedRecordResourceTypeTransaction,
		DeletedRecordResourceTypeTransactionAllocation,
		DeletedRecordResourceTypeUnit,
		DeletedRecordResourceTypeUnitGroup,
		DeletedRecordResourceTypeUnitGroupUnit,
		DeletedRecordResourceTypeVolumeDiscount:
		return true
	default:
		return false
	}
}

func (m DeletedRecordResourceType) EnumValues() []string {
	return []string{
		string(DeletedRecordResourceTypeAccountGroup),
		string(DeletedRecordResourceTypeAccountGroupProductLineAccess),
		string(DeletedRecordResourceTypeAccountIntegration),
		string(DeletedRecordResourceTypeAccountPrice),
		string(DeletedRecordResourceTypeAccountUser),
		string(DeletedRecordResourceTypeAddress),
		string(DeletedRecordResourceTypeAttribute),
		string(DeletedRecordResourceTypeBatch),
		string(DeletedRecordResourceTypeCarrier),
		string(DeletedRecordResourceTypeServiceLevel),
		string(DeletedRecordResourceTypeConsumption),
		string(DeletedRecordResourceTypeCustomAgent),
		string(DeletedRecordResourceTypeAgentMemory),
		string(DeletedRecordResourceTypeMessagingGroup),
		string(DeletedRecordResourceTypeCustomer),
		string(DeletedRecordResourceTypeCustomerProductLineAccess),
		string(DeletedRecordResourceTypeDCLocation),
		string(DeletedRecordResourceTypeDepartment),
		string(DeletedRecordResourceTypeItemCategory),
		string(DeletedRecordResourceTypeMachine),
		string(DeletedRecordResourceTypeMaterial),
		string(DeletedRecordResourceTypeOrderDiscount),
		string(DeletedRecordResourceTypePart),
		string(DeletedRecordResourceTypePaymentTerm),
		string(DeletedRecordResourceTypeProduct),
		string(DeletedRecordResourceTypeProductLine),
		string(DeletedRecordResourceTypeProductType),
		string(DeletedRecordResourceTypeProductionRun),
		string(DeletedRecordResourceTypeProductionStep),
		string(DeletedRecordResourceTypeProperty),
		string(DeletedRecordResourceTypePurchaseOrder),
		string(DeletedRecordResourceTypePurchaseOrderLine),
		string(DeletedRecordResourceTypeRegistrationFlow),
		string(DeletedRecordResourceTypeRole),
		string(DeletedRecordResourceTypeSalesOrder),
		string(DeletedRecordResourceTypeSalesOrderLine),
		string(DeletedRecordResourceTypeSandbox),
		string(DeletedRecordResourceTypeScanningStation),
		string(DeletedRecordResourceTypeSettlement),
		string(DeletedRecordResourceTypeShipment),
		string(DeletedRecordResourceTypeShipmentLine),
		string(DeletedRecordResourceTypeShippingCase),
		string(DeletedRecordResourceTypeShippingTerm),
		string(DeletedRecordResourceTypeLocation),
		string(DeletedRecordResourceTypeSupplier),
		string(DeletedRecordResourceTypeSupplierMaterial),
		string(DeletedRecordResourceTypeTerritory),
		string(DeletedRecordResourceTypeTransaction),
		string(DeletedRecordResourceTypeTransactionAllocation),
		string(DeletedRecordResourceTypeUnit),
		string(DeletedRecordResourceTypeUnitGroup),
		string(DeletedRecordResourceTypeUnitGroupUnit),
		string(DeletedRecordResourceTypeVolumeDiscount),
	}
}
