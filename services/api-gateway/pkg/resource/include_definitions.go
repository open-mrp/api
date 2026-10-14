package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// includeRegistry is every include registration made here, by object type, for the adherence test.
var includeRegistry = map[constants.ObjectType]*apiendpoint.ObjectIncludes{}

func registerIncludes(oi *apiendpoint.ObjectIncludes) {
	includeRegistry[oi.ObjectType] = oi
	apiendpoint.RegisterIncludes(oi)
}

func init() {
	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeCustomerPricingFinding,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
			{Key: "unit_price.numerator_unit", ObjectType: constants.ObjectTypeUnit},
			{Key: "unit_price.denominator_unit", ObjectType: constants.ObjectTypeUnit},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeRealizedMarginFinding,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "customer_group", ObjectType: constants.ObjectTypeAccountGroup},
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeRole,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "permissions", ObjectType: constants.ObjectTypeRolePermission},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAgentDefinition,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "config", ObjectType: constants.ObjectTypeAgentDefinition},
			{Key: "tools", ObjectType: constants.ObjectTypeAgentDefinitionTool},
			{Key: "role", ObjectType: constants.ObjectTypeRole},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAgentRun,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "triggered_by", ObjectType: constants.ObjectTypeActor},
			{Key: "actions", ObjectType: constants.ObjectTypeAgentAction},
			{Key: "definition", ObjectType: constants.ObjectTypeAgentDefinition},
			{Key: "steps", ObjectType: constants.ObjectTypeAgentRunStep},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeNotification,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "sender", ObjectType: constants.ObjectTypeActor},
			{Key: "resource", ObjectType: constants.ObjectTypeEntity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAnnouncement,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "resource", ObjectType: constants.ObjectTypeEntity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAPIKey,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "role", ObjectType: constants.ObjectTypeRole},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeActor,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "role", ObjectType: constants.ObjectTypeRole},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeRequestLog,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "account", ObjectType: constants.ObjectTypeAccount},
			{Key: "query_params", ObjectType: constants.ObjectTypeRequestLog},
			{Key: "request_body", ObjectType: constants.ObjectTypeRequestLog},
			{Key: "response_body", ObjectType: constants.ObjectTypeRequestLog},
			{Key: "actor", ObjectType: constants.ObjectTypeActor, Children: []apiendpoint.IncludeFieldDef{
				{Key: "role", ObjectType: constants.ObjectTypeRole},
			}},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAuditEvent,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "actor", ObjectType: constants.ObjectTypeActor},
			{Key: "account", ObjectType: constants.ObjectTypeAccount},
			{Key: "changes", ObjectType: constants.ObjectTypeAuditEvent},
			{Key: "metadata", ObjectType: constants.ObjectTypeAuditEvent},
			{Key: "request", ObjectType: constants.ObjectTypeRequestLog},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeEmailLog,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "sent_by", ObjectType: constants.ObjectTypeActor},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeSandbox,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner_account", ObjectType: constants.ObjectTypeAccount},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeToolGroup,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "tools", ObjectType: constants.ObjectTypeAvailableTool},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAccountUser,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "user", ObjectType: constants.ObjectTypeUser},
			{Key: "role", ObjectType: constants.ObjectTypeRole},
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeContactMatch,
		Fields: []apiendpoint.IncludeFieldDef{
			{
				Key:        "account_user",
				ObjectType: constants.ObjectTypeAccountUser,
				Children: []apiendpoint.IncludeFieldDef{
					{Key: "user", ObjectType: constants.ObjectTypeUser},
					{Key: "role", ObjectType: constants.ObjectTypeRole},
					{Key: "department", ObjectType: constants.ObjectTypeDepartment},
				},
			},
			{Key: "account", ObjectType: constants.ObjectTypeAccount},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeOrderNotificationRecipient,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "account_user", ObjectType: constants.ObjectTypeAccountUser},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeMessagingBlock,
		Fields: []apiendpoint.IncludeFieldDef{
			{
				Key:        "blocked_user",
				ObjectType: constants.ObjectTypeAccountUser,
				Children: []apiendpoint.IncludeFieldDef{
					{Key: "user", ObjectType: constants.ObjectTypeUser},
					{Key: "role", ObjectType: constants.ObjectTypeRole},
					{Key: "department", ObjectType: constants.ObjectTypeDepartment},
				},
			},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeDepartment,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "location", ObjectType: constants.ObjectTypeLocation},
			{Key: "scanning_stations", ObjectType: constants.ObjectTypeScanningStation},
			{Key: "machines", ObjectType: constants.ObjectTypeMachine},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeConsumption,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "waste_quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "consumed_item", ObjectType: constants.ObjectTypeItem},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeCustomer,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "bill_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "ship_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "type", ObjectType: constants.ObjectTypeAccountGroup},
			{Key: "parent_account", ObjectType: constants.ObjectTypeCustomer},
			{
				Key:        "freight_preferences",
				ObjectType: constants.ObjectTypeCustomer,
				Children: []apiendpoint.IncludeFieldDef{
					{Key: "carrier", ObjectType: constants.ObjectTypeCarrier},
					{Key: "service_level", ObjectType: constants.ObjectTypeServiceLevel},
				},
			},
			{
				Key:        "defaults",
				ObjectType: constants.ObjectTypeCustomer,
				Children: []apiendpoint.IncludeFieldDef{
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
			{Key: "credit_limit", ObjectType: constants.ObjectTypeQuantity, Children: []apiendpoint.IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeQuantity,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "unit", ObjectType: constants.ObjectTypeUnit},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeInventoryItem,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeItemLotDefault,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "unit", ObjectType: constants.ObjectTypeUnit},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeCustomerLeadTime,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "account_group", ObjectType: constants.ObjectTypeAccountGroup},
			{Key: "parent_customer", ObjectType: constants.ObjectTypeCustomer},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeItemInventory,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "on_hand", ObjectType: constants.ObjectTypeComputedQuantity},
			{Key: "reserved", ObjectType: constants.ObjectTypeComputedQuantity},
			{Key: "available_to_promise", ObjectType: constants.ObjectTypeComputedQuantity},
			{Key: "short", ObjectType: constants.ObjectTypeComputedQuantity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeShippingTerm,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "flat_rate", ObjectType: constants.ObjectTypeQuantity},
			{
				Key:        "minimum_order_value",
				ObjectType: constants.ObjectTypeQuantity,
				Children: []apiendpoint.IncludeFieldDef{
					{Key: "unit", ObjectType: constants.ObjectTypeUnit},
				},
			},
			{Key: "free_shipping_service_levels", ObjectType: constants.ObjectTypeServiceLevel},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeCarrier,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "service_levels", ObjectType: constants.ObjectTypeServiceLevel},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAccountPrice,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "recipient_account", ObjectType: constants.ObjectTypeAccount},
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
			{Key: "categories", ObjectType: constants.ObjectTypeItemCategory},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeProperty,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAccount,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "default_billing_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "default_shipping_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "branding", ObjectType: constants.ObjectTypeAccountBranding},
			{Key: "portal", ObjectType: constants.ObjectTypeAccountPortal},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeItem,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "category", ObjectType: constants.ObjectTypeItemCategory},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
			{Key: "unit_value", ObjectType: constants.ObjectTypeRate},
			{Key: "unit_cost", ObjectType: constants.ObjectTypeRate},
			{Key: "burn_rate", ObjectType: constants.ObjectTypeRate},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePart,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "category", ObjectType: constants.ObjectTypeItemCategory},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
			{Key: "unit_value", ObjectType: constants.ObjectTypeRate},
			{Key: "unit_cost", ObjectType: constants.ObjectTypeRate},
			{Key: "burn_rate", ObjectType: constants.ObjectTypeRate},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeItemCategory,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "properties", ObjectType: constants.ObjectTypeProperty},
			{Key: "unit_group", ObjectType: constants.ObjectTypeUnitGroup},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeUnitGroup,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "base_unit", ObjectType: constants.ObjectTypeUnit},
			{Key: "associated_units", ObjectType: constants.ObjectTypeUnitGroupUnit},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeUnitGroupUnit,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "unit", ObjectType: constants.ObjectTypeUnit},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeSalesOrder,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "sales_rep", ObjectType: constants.ObjectTypeActor},
			{Key: "created_by", ObjectType: constants.ObjectTypeCreatedBy},
			{Key: "bill_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "ship_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "freight", ObjectType: constants.ObjectTypeFreight},
			{Key: "payment_term", ObjectType: constants.ObjectTypePaymentTerm},
			{Key: "shipping_term", ObjectType: constants.ObjectTypeShippingTerm},
			{Key: "order_discount", ObjectType: constants.ObjectTypeOrderDiscount},
			{Key: "totals", ObjectType: constants.ObjectTypeSalesOrderTotals},
			{Key: "contacts", ObjectType: constants.ObjectTypeOrderContact},
			{
				Key:        "related",
				ObjectType: constants.ObjectTypeSalesOrderRelated,
				Children: []apiendpoint.IncludeFieldDef{
					{Key: "pick", ObjectType: constants.ObjectTypeRecord},
					{Key: "production_run", ObjectType: constants.ObjectTypeRecord},
					{Key: "shipments", ObjectType: constants.ObjectTypeRecord},
					{Key: "invoices", ObjectType: constants.ObjectTypeRecord},
				},
			},
			{Key: "lines", ObjectType: constants.ObjectTypeSalesOrderLine},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeSalesOrderLine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "product", ObjectType: constants.ObjectTypeProduct},
			{Key: "quantity_ordered", ObjectType: constants.ObjectTypeQuantity},
			{Key: "unit_price", ObjectType: constants.ObjectTypeRate},
			{Key: "unit_cost", ObjectType: constants.ObjectTypeRate},
			{Key: "totals", ObjectType: constants.ObjectTypeSalesOrderTotals},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeInvoice,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "order", ObjectType: constants.ObjectTypeSalesOrder},
			{Key: "shipment", ObjectType: constants.ObjectTypeShipment},
			{Key: "billing_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "payment_term", ObjectType: constants.ObjectTypePaymentTerm},
			{Key: "lines", ObjectType: constants.ObjectTypeInvoiceLine},
			{Key: "allocations", ObjectType: constants.ObjectTypeInvoiceAllocation},
			{
				Key:        "related",
				ObjectType: constants.ObjectTypeInvoiceRelated,
				Children: []apiendpoint.IncludeFieldDef{
					{Key: "sales_order", ObjectType: constants.ObjectTypeRecord},
					{Key: "shipment", ObjectType: constants.ObjectTypeRecord},
				},
			},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeInvoiceForPayment,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "parent_account", ObjectType: constants.ObjectTypeAccount},
			{Key: "billing_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "allocations", ObjectType: constants.ObjectTypeInvoiceAllocation},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeInvoiceLine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "order_line", ObjectType: constants.ObjectTypeSalesOrderLine},
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			// The quantity and the price are always on the line; naming them here is what lets a caller reach through to the units they are counted in.
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "unit_price", ObjectType: constants.ObjectTypeRate},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeInvoiceAllocation,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "transaction", ObjectType: constants.ObjectTypeTransaction},
			// The amount is always on the allocation; naming it here is what lets a caller reach through to the currency it is counted in.
			{Key: "amount", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeTransactionAllocation,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "transaction", ObjectType: constants.ObjectTypeTransaction},
			// The amount is always on the allocation; naming it here is what lets a caller reach through to the currency it is counted in.
			{Key: "amount", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeDelivery,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "related", ObjectType: constants.ObjectTypeDeliveryRelated},
			{Key: "lines", ObjectType: constants.ObjectTypeDeliveryLine},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeDeliveryLine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "order_line", ObjectType: constants.ObjectTypePurchaseOrderLine},
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			// The quantity is always on the line; naming it here is what lets a caller reach through it to the unit it is counted in.
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "unit_cost", ObjectType: constants.ObjectTypeRate},
			{Key: "location", ObjectType: constants.ObjectTypeLocation},
			{Key: "lot", ObjectType: constants.ObjectTypeLot},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeDeliveryRelated,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "purchase_order", ObjectType: constants.ObjectTypeRecord},
			{Key: "receiving_order", ObjectType: constants.ObjectTypeRecord},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeReceivingOrder,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "supplier", ObjectType: constants.ObjectTypeSupplier},
			{Key: "lines", ObjectType: constants.ObjectTypeReceivingOrderLine},
			{Key: "totals", ObjectType: constants.ObjectTypeReceivingOrderTotals},
			{Key: "related", ObjectType: constants.ObjectTypeReceivingOrderRelated},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeReceivingOrderRelated,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "purchase_order", ObjectType: constants.ObjectTypeRecord},
			{Key: "deliveries", ObjectType: constants.ObjectTypeRecord},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeReceivingOrderLine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "order_line", ObjectType: constants.ObjectTypePurchaseOrderLine},
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			// Both quantities are always on the line; naming them here is what lets a caller reach through to the units they are counted in. The rejected figure needs no entry — it is computed, and arrives with its unit already.
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "quantity_ordered", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeInventoryChangeLog,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "responsible_user", ObjectType: constants.ObjectTypeUser},
			{Key: "responsible_scanning_station", ObjectType: constants.ObjectTypeScanningStation},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeMachine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeMachineItemScanSuggestion,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeMachineDowntimeEvent,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "machine", ObjectType: constants.ObjectTypeMachine},
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "reported_by", ObjectType: constants.ObjectTypeActor},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeDemandOverride,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "scope", ObjectType: constants.ObjectTypeEntity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeJob,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "created_by", ObjectType: constants.ObjectTypeActor},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePick,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "created_by", ObjectType: constants.ObjectTypeCreatedBy},
			{Key: "freight", ObjectType: constants.ObjectTypeFreight},
			{Key: "lines", ObjectType: constants.ObjectTypePickLine},
			{
				Key:        "related",
				ObjectType: constants.ObjectTypePickRelated,
				Children: []apiendpoint.IncludeFieldDef{
					{Key: "sales_order", ObjectType: constants.ObjectTypeRecord},
					{Key: "shipments", ObjectType: constants.ObjectTypeRecord},
				},
			},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePickLine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			{Key: "sales_order_line", ObjectType: constants.ObjectTypeSalesOrderLine},
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity, Children: []apiendpoint.IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
			{Key: "ordered_quantity", ObjectType: constants.ObjectTypeQuantity, Children: []apiendpoint.IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeMaterial,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeProductLine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
			{Key: "unit_group", ObjectType: constants.ObjectTypeUnitGroup},
			{Key: "default_lot", ObjectType: constants.ObjectTypeQuantity, Children: []apiendpoint.IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeProduct,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "product_type", ObjectType: constants.ObjectTypeProductType},
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
			{Key: "item", ObjectType: constants.ObjectTypeItem},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePurchaseOrder,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "supplier", ObjectType: constants.ObjectTypeSupplier},
			{Key: "created_by", ObjectType: constants.ObjectTypeCreatedBy},
			{Key: "bill_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "ship_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "freight", ObjectType: constants.ObjectTypeFreight},
			{Key: "payment_term", ObjectType: constants.ObjectTypePaymentTerm},
			{Key: "shipping_term", ObjectType: constants.ObjectTypeShippingTerm},
			{Key: "related", ObjectType: constants.ObjectTypePurchaseOrderRelated},
			{Key: "lines", ObjectType: constants.ObjectTypePurchaseOrderLine},
			{Key: "contacts", ObjectType: constants.ObjectTypeEmailContact},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePurchaseOrderLine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			// The quantity and the two rates are always on the line; naming them here is what lets a caller reach through to the units they are counted in.
			{Key: "quantity_ordered", ObjectType: constants.ObjectTypeQuantity},
			{Key: "unit_price", ObjectType: constants.ObjectTypeRate},
			{Key: "delivery_lines", ObjectType: constants.ObjectTypeDeliveryLine},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePurchaseOrderRelated,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "receiving_order", ObjectType: constants.ObjectTypeRecord},
			{Key: "deliveries", ObjectType: constants.ObjectTypeRecord},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeTerritory,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "sales_rep", ObjectType: constants.ObjectTypeAccountUser},
			{Key: "product_line", ObjectType: constants.ObjectTypeProductLine},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeSupplier,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "bill_to_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "ship_to_address", ObjectType: constants.ObjectTypeAddress},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeLocation,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "parent", ObjectType: constants.ObjectTypeLocation},
			{Key: "children", ObjectType: constants.ObjectTypeLocation},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeScanningStation,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
			{Key: "production_steps", ObjectType: constants.ObjectTypeProductionStep},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeProductionRun,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "responsible_user", ObjectType: constants.ObjectTypeAccountUser},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeVolumeDiscount,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "customer_groups", ObjectType: constants.ObjectTypeAccountGroup},
			{Key: "product_lines", ObjectType: constants.ObjectTypeProductLine},
			{Key: "categories", ObjectType: constants.ObjectTypeItemCategory},
			{Key: "attributes", ObjectType: constants.ObjectTypeAttribute},
			{Key: "acceptable_units", ObjectType: constants.ObjectTypeUnit},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeShippingCase,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "carrier", ObjectType: constants.ObjectTypeCarrier},
			{Key: "shipment", ObjectType: constants.ObjectTypeShipment},
			{Key: "freight_amount", ObjectType: constants.ObjectTypeQuantity, Children: []apiendpoint.IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
			{Key: "freight_weight", ObjectType: constants.ObjectTypeQuantity, Children: []apiendpoint.IncludeFieldDef{
				{Key: "unit", ObjectType: constants.ObjectTypeUnit},
			}},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeShipmentLine,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "sales_order_line", ObjectType: constants.ObjectTypeSalesOrderLine},
			{Key: "item", ObjectType: constants.ObjectTypeItem},
			// The quantity is always on the line; naming it here is what lets a caller reach through it to the unit it is counted in.
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeShipment,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "lines", ObjectType: constants.ObjectTypeShipmentLine},
			{Key: "shipping_cases", ObjectType: constants.ObjectTypeShippingCase},
			{Key: "freight", ObjectType: constants.ObjectTypeFreight},
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "shipping_address", ObjectType: constants.ObjectTypeAddress},
			{Key: "shipped_by", ObjectType: constants.ObjectTypeCreatedBy},
			{
				Key:        "related",
				ObjectType: constants.ObjectTypeShipmentRelated,
				Children: []apiendpoint.IncludeFieldDef{
					{Key: "sales_order", ObjectType: constants.ObjectTypeRecord},
					{Key: "pick", ObjectType: constants.ObjectTypeRecord},
					{Key: "invoice", ObjectType: constants.ObjectTypeRecord},
				},
			},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeTransaction,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "allocations", ObjectType: constants.ObjectTypeTransactionAllocation},
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
			{Key: "responsible_user", ObjectType: constants.ObjectTypeAccountUser},
			// The amount is always on the transaction, so it is never requested on its own — an
			// endpoint that returns a transaction directly offers no `amount` key. It is named here
			// only so a caller reaching a transaction through an allocation can go on to the
			// currency the amount is counted in.
			{Key: "amount", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	// The transactions LIST returns transaction_summary (distinct from the detail).
	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeTransactionSummary,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "customer", ObjectType: constants.ObjectTypeCustomer},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeSettlement,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "responsible_user", ObjectType: constants.ObjectTypeAccountUser},
			{Key: "allocations", ObjectType: constants.ObjectTypeTransactionAllocation},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeBatch,
		Fields: []apiendpoint.IncludeFieldDef{
			// The three measures are always on the batch; naming them here is what lets a caller reach through to the unit each is counted in.
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
			{Key: "seconds", ObjectType: constants.ObjectTypeQuantity},
			{Key: "waste", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeBatchFlowNode,
		Fields: []apiendpoint.IncludeFieldDef{
			// The batch is already on the node; naming it is what lets a caller reach through to its measures' units.
			{Key: "batch", ObjectType: constants.ObjectTypeBatch},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeRate,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "numerator_unit", ObjectType: constants.ObjectTypeUnit},
			{Key: "denominator_unit", ObjectType: constants.ObjectTypeUnit},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeSupplierMaterial,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "material", ObjectType: constants.ObjectTypeMaterial},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeOwner,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "account", ObjectType: constants.ObjectTypeAccount},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeUnit,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeServiceLevel,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePaymentTerm,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePriority,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAccountStatus,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeSalesOrderStatus,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAdjustmentType,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypePermissionGroup,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "owner", ObjectType: constants.ObjectTypeOwner},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeProduction,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "produced_item", ObjectType: constants.ObjectTypeItem},
			// The quantity is always on the production; naming it lets a caller reach its unit.
			{Key: "quantity", ObjectType: constants.ObjectTypeQuantity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeProductionStep,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "production", ObjectType: constants.ObjectTypeProduction},
			{Key: "consumptions", ObjectType: constants.ObjectTypeConsumption},
			{Key: "machines", ObjectType: constants.ObjectTypeMachine},
			{Key: "scanning_station", ObjectType: constants.ObjectTypeScanningStation},
			{Key: "department", ObjectType: constants.ObjectTypeDepartment},
			{Key: "in_steps", ObjectType: constants.ObjectTypeProductionStep},
			{Key: "out_steps", ObjectType: constants.ObjectTypeProductionStep},
			// The rates are always on the step; naming them here is what lets a caller reach through
			// to the units they are counted in.
			{Key: "labor_rate", ObjectType: constants.ObjectTypeRate},
			{Key: "labor_time", ObjectType: constants.ObjectTypeRate},
			{Key: "overhead_rate", ObjectType: constants.ObjectTypeRate},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeProductionFlow,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "steps", ObjectType: constants.ObjectTypeProductionStep},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeConversation,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "assignee", ObjectType: constants.ObjectTypeActor},
			{Key: "group", ObjectType: constants.ObjectTypeMessagingGroup},
			{Key: "participants", ObjectType: constants.ObjectTypeConversationParticipant},
			{Key: "topic", ObjectType: constants.ObjectTypeEntity},
			{Key: "last_message", ObjectType: constants.ObjectTypeChatMessage},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeConversationLink,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "conversation", ObjectType: constants.ObjectTypeConversation},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeChatMessage,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "sender", ObjectType: constants.ObjectTypeActor},
			{Key: "author", ObjectType: constants.ObjectTypeActor},
			{Key: "resource", ObjectType: constants.ObjectTypeEntity},
			{Key: "attachments", ObjectType: constants.ObjectTypeMessageAttachment},
			{Key: "conversation", ObjectType: constants.ObjectTypeConversation},
			{Key: "reply_to", ObjectType: constants.ObjectTypeChatMessage},
			{Key: "agent_run", ObjectType: constants.ObjectTypeAgentRun},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeMessageAttachment,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "resource", ObjectType: constants.ObjectTypeEntity},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeAttachmentUploadTarget,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "attachment", ObjectType: constants.ObjectTypeMessageAttachment},
		},
	})

	registerIncludes(&apiendpoint.ObjectIncludes{
		ObjectType: constants.ObjectTypeEmailInbox,
		Fields: []apiendpoint.IncludeFieldDef{
			{Key: "email_domain", ObjectType: constants.ObjectTypeEmailDomain},
			{Key: "agent_config", ObjectType: constants.ObjectTypeAgentDefinition},
		},
	})
}
