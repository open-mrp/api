package apiendpoint

import "github.com/augno/api/shared/constants"

func init() {
	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeRole,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "permissions", ObjectType: constants.ObjectTypeRolePermission},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAgentDefinition,
		Fields: []IncludeFieldDef{
			{Key: "config", ObjectType: constants.ObjectTypeAgentDefinition},
			{Key: "tools", ObjectType: constants.ObjectTypeAgentDefinitionTool},
			{Key: "role", ObjectType: constants.ObjectTypeRole},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAgentAlert,
		Fields: []IncludeFieldDef{
			{Key: "run", ObjectType: constants.ObjectTypeAgentRun},
			{Key: "action", ObjectType: constants.ObjectTypeAgentAction},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAgentRun,
		Fields: []IncludeFieldDef{
			{Key: "actions", ObjectType: constants.ObjectTypeAgentAction},
			{Key: "definition", ObjectType: constants.ObjectTypeAgentDefinition},
			{Key: "steps", ObjectType: constants.ObjectTypeAgentRunStep},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAPIKey,
		Fields: []IncludeFieldDef{
			{Key: "role", ObjectType: constants.ObjectTypeRole},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeActor,
		Fields: []IncludeFieldDef{
			{Key: "role", ObjectType: constants.ObjectTypeRole},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeRequestLog,
		Fields: []IncludeFieldDef{
			{Key: "account", ObjectType: constants.ObjectTypeAccount},
			{Key: "query_params", ObjectType: constants.ObjectTypeRequestLog},
			{Key: "request_body", ObjectType: constants.ObjectTypeRequestLog},
			{Key: "response_body", ObjectType: constants.ObjectTypeRequestLog},
			{Key: "actor", ObjectType: constants.ObjectTypeActor, Children: []IncludeFieldDef{
				{Key: "role", ObjectType: constants.ObjectTypeRole},
			}},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAuditEvent,
		Fields: []IncludeFieldDef{
			{Key: "actor", ObjectType: constants.ObjectTypeActor},
			{Key: "changes", ObjectType: constants.ObjectTypeAuditEvent},
			{Key: "metadata", ObjectType: constants.ObjectTypeAuditEvent},
			{Key: "request", ObjectType: constants.ObjectTypeRequestLog},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeEmailLog,
		Fields: []IncludeFieldDef{
			{Key: "sent_by", ObjectType: constants.ObjectTypeActor},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeSandbox,
		Fields: []IncludeFieldDef{
			{Key: "owner_account", ObjectType: constants.ObjectTypeAccount},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeToolGroup,
		Fields: []IncludeFieldDef{
			{Key: "tools", ObjectType: constants.ObjectTypeAvailableTool},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAccountUser,
		Fields: []IncludeFieldDef{
			{Key: "user", ObjectType: constants.ObjectTypeUser},
			{Key: "role", ObjectType: constants.ObjectTypeRole},
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeDepartment,
		Fields: []IncludeFieldDef{
			{Key: "location", ObjectType: constants.ObjectTypeLocation},
			{Key: "scanning_stations", ObjectType: constants.ObjectTypeScanningStation},
			{Key: "machines", ObjectType: constants.ObjectTypeMachine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeConsumption,
		Fields: []IncludeFieldDef{
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "waste_quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "consumed_item", ObjectType: constants.ObjectTypeItem},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeCustomer,
		Fields: []IncludeFieldDef{
			{Key: "bill_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "ship_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "type", ObjectType: constants.ObjectTypeAccountGroup},
			{Key: "parent_account", ObjectType: constants.ObjectTypeCustomer},
			{
				Key:        "freight_preferences",
				ObjectType: constants.ObjectTypeCustomer,
				Children: []IncludeFieldDef{
					{Key: "carrier", ObjectType: constants.ObjectTypeCarrier},
					{Key: "service_level", ObjectType: constants.ObjectTypeServiceLevel},
				},
			},
			{
				Key:        "defaults",
				ObjectType: constants.ObjectTypeCustomer,
				Children: []IncludeFieldDef{
					{Key: "payment_term", ObjectType: constants.ObjectTypePaymentTerm},
					{Key: "shipping_term", ObjectType: constants.ObjectTypeShippingTerm},
					{Key: "priority", ObjectType: constants.ObjectTypePriority},
					{Key: "sales_rep", ObjectType: constants.ObjectTypeAccountUser},
				},
			},
			{Key: "contact_info", ObjectType: constants.ObjectTypeCustomerContactInfo},
			{Key: "notification_preferences", ObjectType: constants.ObjectTypeCustomerNotificationPreferences},
			{Key: "price_groups", ObjectType: constants.ObjectTypeAccountGroup},
			{Key: "child_accounts", ObjectType: constants.ObjectTypeCustomer},
			{Key: "credit_limit", ObjectType: constants.ObjectTypeQuantity, Children: []IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeQuantity,
		Fields: []IncludeFieldDef{
			{Key: "unit", ObjectType: constants.ObjectTypeUnit},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeInventoryItem,
		Fields: []IncludeFieldDef{
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeItemInventory,
		Fields: []IncludeFieldDef{
			{Key: "on_hand", ObjectType: constants.ObjectTypeQuantity},
			{Key: "reserved", ObjectType: constants.ObjectTypeQuantity},
			{Key: "available_to_promise", ObjectType: constants.ObjectTypeQuantity},
			{Key: "short", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeShippingTerm,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "flat_rate", ObjectType: constants.ObjectTypeQuantity},
			{
				Key:        "minimum_order_value",
				ObjectType: constants.ObjectTypeQuantity,
				Children: []IncludeFieldDef{
					{Key: "unit", ObjectType: constants.ObjectTypeUnit},
				},
			},
			{Key: "free_shipping_service_levels", ObjectType: constants.ObjectTypeServiceLevel},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeCarrier,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "service_levels", ObjectType: constants.ObjectTypeServiceLevel},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAccountPrice,
		Fields: []IncludeFieldDef{
			{Key: "recipient_account", ObjectType: constants.ObjectTypeAccount},
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
			{Key: "categories", ObjectType: constants.ObjectTypeItemCategory},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeProperty,
		Fields: []IncludeFieldDef{
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAccount,
		Fields: []IncludeFieldDef{
			{Key: "default_billing_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "default_shipping_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "branding", ObjectType: constants.ObjectTypeAccountBranding},
			{Key: "portal", ObjectType: constants.ObjectTypeAccountPortal},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeItem,
		Fields: []IncludeFieldDef{
			{Key: "category", ObjectType: constants.ObjectTypeItemCategory},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
			{Key: "unit_value", ObjectType: constants.ObjectTypeRate},
			{Key: "unit_cost", ObjectType: constants.ObjectTypeRate},
			{Key: "burn_rate", ObjectType: constants.ObjectTypeRate},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypePart,
		Fields: []IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "category", ObjectType: constants.ObjectTypeItemCategory},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
			{Key: "unit_value", ObjectType: constants.ObjectTypeRate},
			{Key: "unit_cost", ObjectType: constants.ObjectTypeRate},
			{Key: "burn_rate", ObjectType: constants.ObjectTypeRate},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeItemCategory,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "properties", ObjectType: constants.ObjectTypeProperty},
			{Key: "unit_group", ObjectType: constants.ObjectTypeUnitGroup},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeUnitGroup,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "base_unit", ObjectType: constants.ObjectTypeUnit},
			{Key: "associated_units", ObjectType: constants.ObjectTypeUnitGroupUnit},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeUnitGroupUnit,
		Fields: []IncludeFieldDef{
			{Key: "unit", ObjectType: constants.ObjectTypeUnit},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeSalesOrder,
		Fields: []IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "sales_rep", ObjectType: constants.ObjectTypeActor},
			{Key: "bill_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "ship_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "freight", ObjectType: constants.ObjectTypeFreight},
			{Key: "payment_term", ObjectType: constants.ObjectTypePaymentTerm},
			{Key: "shipping_term", ObjectType: constants.ObjectTypeShippingTerm},
			{Key: "order_discount", ObjectType: constants.ObjectTypeOrderDiscount},
			{Key: "totals", ObjectType: constants.ObjectTypeSalesOrderTotals},
			{
				Key:        "related",
				ObjectType: constants.ObjectTypeSalesOrderRelated,
				Children: []IncludeFieldDef{
					{Key: "pick", ObjectType: constants.ObjectTypeRecord},
					{Key: "production_run", ObjectType: constants.ObjectTypeRecord},
					{Key: "shipments", ObjectType: constants.ObjectTypeRecord},
				},
			},
			{Key: "lines", ObjectType: constants.ObjectTypeSalesOrderLine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeSalesOrderLine,
		Fields: []IncludeFieldDef{
			{Key: "product", ObjectType: constants.ObjectTypeProduct},
			{Key: "quantity_ordered", ObjectType: constants.ObjectTypeQuantity},
			{Key: "unit_price", ObjectType: constants.ObjectTypeRate},
			{Key: "unit_cost", ObjectType: constants.ObjectTypeRate},
			{Key: "totals", ObjectType: constants.ObjectTypeSalesOrderTotals},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeInvoice,
		Fields: []IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "order", ObjectType: constants.ObjectTypeSalesOrder},
			{Key: "billing_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "shipment", ObjectType: constants.ObjectTypeShipment},
			{Key: "payment_term", ObjectType: constants.ObjectTypePaymentTerm},
			{Key: "lines", ObjectType: constants.ObjectTypeInvoiceLine},
			{Key: "allocations", ObjectType: constants.ObjectTypeInvoiceAllocation},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeInvoiceForPayment,
		Fields: []IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "parent_account", ObjectType: constants.ObjectTypeAccount},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeInvoiceLine,
		Fields: []IncludeFieldDef{
			{Key: "order_line", ObjectType: constants.ObjectTypeSalesOrderLine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeInvoiceAllocation,
		Fields: []IncludeFieldDef{
			{Key: "transaction", ObjectType: constants.ObjectTypeTransaction},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeTransactionAllocation,
		Fields: []IncludeFieldDef{
			{Key: "transaction", ObjectType: constants.ObjectTypeTransaction},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeDelivery,
		Fields: []IncludeFieldDef{
			{Key: "purchase_order", ObjectType: constants.ObjectTypePurchaseOrder},
			{Key: "lines", ObjectType: constants.ObjectTypeDeliveryLine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeReceivingOrder,
		Fields: []IncludeFieldDef{
			{Key: "purchase_order", ObjectType: constants.ObjectTypePurchaseOrder},
			{Key: "supplier", ObjectType: constants.ObjectTypeAccount},
			{Key: "lines", ObjectType: constants.ObjectTypeReceivingOrderLine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeReceivingOrderLine,
		Fields: []IncludeFieldDef{
			{Key: "order_line", ObjectType: constants.ObjectTypeSalesOrderLine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeInventoryChangeLog,
		Fields: []IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "responsible_user", ObjectType: constants.ObjectTypeUser},
			{Key: "responsible_scanning_station", ObjectType: constants.ObjectTypeScanningStation},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeMachine,
		Fields: []IncludeFieldDef{
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypePick,
		Fields: []IncludeFieldDef{
			{Key: "sales_order", ObjectType: constants.ObjectTypeSalesOrder},
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "departments", ObjectType: constants.ObjectTypeDepartment},
			{Key: "lines", ObjectType: constants.ObjectTypePickLine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypePickLine,
		Fields: []IncludeFieldDef{
			{Key: "sales_order_line", ObjectType: constants.ObjectTypeSalesOrderLine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeMaterial,
		Fields: []IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeProductLine,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "unit_group", ObjectType: constants.ObjectTypeUnitGroup},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeProduct,
		Fields: []IncludeFieldDef{
			{Key: "product_type", ObjectType: constants.ObjectTypeProductType},
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
			{Key: "item", ObjectType: constants.ObjectTypeItem},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypePurchaseOrder,
		Fields: []IncludeFieldDef{
			{Key: "supplier", ObjectType: constants.ObjectTypeSupplier},
			{Key: "bill_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "ship_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "freight", ObjectType: constants.ObjectTypeFreight},
			{Key: "payment_term", ObjectType: constants.ObjectTypePaymentTerm},
			{Key: "shipping_term", ObjectType: constants.ObjectTypeShippingTerm},
			{Key: "receiving_order", ObjectType: constants.ObjectTypeReceivingOrder},
			{Key: "lines", ObjectType: constants.ObjectTypePurchaseOrderLine},
			{Key: "contacts", ObjectType: constants.ObjectTypeEmailContact},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeTerritory,
		Fields: []IncludeFieldDef{
			{Key: "sales_rep", ObjectType: constants.ObjectTypeAccountUser},
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeSupplier,
		Fields: []IncludeFieldDef{
			{Key: "bill_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "ship_to_address", ObjectType: constants.ObjectTypeAddress},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeLocation,
		Fields: []IncludeFieldDef{
			{Key: "parent", ObjectType: constants.ObjectTypeLocation},
			{Key: "children", ObjectType: constants.ObjectTypeLocation},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeScanningStation,
		Fields: []IncludeFieldDef{
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
			{Key: "production_steps", ObjectType: constants.ObjectTypeProductionStep},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeProductionRun,
		Fields: []IncludeFieldDef{
			{Key: "responsible_user", ObjectType: constants.ObjectTypeAccountUser},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeVolumeDiscount,
		Fields: []IncludeFieldDef{
			{Key: "customer_groups", ObjectType: constants.ObjectTypeAccountGroup},
			{Key: "product_lines", ObjectType: constants.ObjectTypeProductLine},
			{Key: "categories", ObjectType: constants.ObjectTypeItemCategory},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
			{Key: "acceptable_units", ObjectType: constants.ObjectTypeUnit},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeDCLocation,
		Fields: []IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeShippingCase,
		Fields: []IncludeFieldDef{
			{Key: "carrier", ObjectType: constants.ObjectTypeCarrier},
			{Key: "shipment", ObjectType: constants.ObjectTypeShipment},
			{Key: "freight_amount", ObjectType: constants.ObjectTypeQuantity, Children: []IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
			{Key: "freight_weight", ObjectType: constants.ObjectTypeQuantity, Children: []IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeShipment,
		Fields: []IncludeFieldDef{
			{Key: "lines", ObjectType: constants.ObjectTypeShipmentLine},
			{Key: "shipping_cases", ObjectType: constants.ObjectTypeShippingCase},
			{Key: "sales_order", ObjectType: constants.ObjectTypeSalesOrder},
			{Key: "freight", ObjectType: constants.ObjectTypeFreight},
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "shipping_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "shipped_by", ObjectType: constants.ObjectTypeAccountUser},
			{Key: "invoice", ObjectType: constants.ObjectTypeInvoice},
			{Key: "pick", ObjectType: constants.ObjectTypePick},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeTransaction,
		Fields: []IncludeFieldDef{
			{Key: "allocations", ObjectType: constants.ObjectTypeAllocationEntry},
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "responsible_user", ObjectType: constants.ObjectTypeAccountUser},
		},
	})

	// The transactions LIST returns transaction_summary (distinct from the detail).
	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeTransactionSummary,
		Fields: []IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeSettlement,
		Fields: []IncludeFieldDef{
			{Key: "responsible_user", ObjectType: constants.ObjectTypeAccountUser},
			{Key: "allocations", ObjectType: constants.ObjectTypeAllocationEntry},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeRate,
		Fields: []IncludeFieldDef{
			{Key: "numerator_unit", ObjectType: constants.ObjectTypeUnit},
			{Key: "denominator_unit", ObjectType: constants.ObjectTypeUnit},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeSupplierMaterial,
		Fields: []IncludeFieldDef{
			{Key: "material", ObjectType: constants.ObjectTypeMaterial},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeOwner,
		Fields: []IncludeFieldDef{
			{Key: "account", ObjectType: constants.ObjectTypeAccount},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeUnit,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeServiceLevel,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypePaymentTerm,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypePriority,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAccountStatus,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeSalesOrderStatus,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeAdjustmentType,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypePermissionGroup,
		Fields: []IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeProduction,
		Fields: []IncludeFieldDef{
			{Key: "produced_item", ObjectType: constants.ObjectTypeItem},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeProductionStep,
		Fields: []IncludeFieldDef{
			{Key: "production", ObjectType: constants.ObjectTypeProduction},
			{Key: "consumptions", ObjectType: constants.ObjectTypeConsumption},
			{Key: "machines", ObjectType: constants.ObjectTypeMachine},
			{Key: "scanning_station", ObjectType: constants.ObjectTypeScanningStation},
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
			{Key: "in_steps", ObjectType: constants.ObjectTypeProductionStep},
			{Key: "out_steps", ObjectType: constants.ObjectTypeProductionStep},
		},
	})

	RegisterIncludes(&ObjectIncludes{
		ObjectType: constants.ObjectTypeProductionFlow,
		Fields: []IncludeFieldDef{
			{Key: "steps", ObjectType: constants.ObjectTypeProductionStep},
		},
	})
}
