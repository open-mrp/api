package constants

import "github.com/open-mrp/apikit/object"

// Object types of the resources forge.1 adds. They are registered here, all at once, so the units building those resources never edit a shared list.
const (
	// ObjectTypeJob indicates that the object is a manufacturing job: one item made to a quantity.
	ObjectTypeJob ObjectType = "job"
	// ObjectTypeWorkOrder indicates that the object is a work order, a release of many jobs for a period.
	ObjectTypeWorkOrder ObjectType = "work_order"
	// ObjectTypeManufacturingMethod indicates that the object is a manufacturing method: the operations and materials that make an item.
	ObjectTypeManufacturingMethod ObjectType = "manufacturing_method"
	// ObjectTypeMethodOperation indicates that the object is one operation of a manufacturing method.
	ObjectTypeMethodOperation ObjectType = "method_operation"
	// ObjectTypeMethodMaterial indicates that the object is one material of a manufacturing method.
	ObjectTypeMethodMaterial ObjectType = "method_material"
	// ObjectTypePriceList indicates that the object is a price list.
	ObjectTypePriceList ObjectType = "price_list"
	// ObjectTypePriceListRule indicates that the object is one rule of a price list.
	ObjectTypePriceListRule ObjectType = "price_list_rule"
	// ObjectTypeCreditNote indicates that the object is a credit note issued to a customer.
	ObjectTypeCreditNote ObjectType = "credit_note"
	// ObjectTypePayment indicates that the object is a payment or refund.
	ObjectTypePayment ObjectType = "payment"
	// ObjectTypeApplication indicates that the object is an application of a credit to a debit.
	ObjectTypeApplication ObjectType = "application"
	// ObjectTypeBill indicates that the object is a bill: a supplier's invoice, read by the buyer.
	ObjectTypeBill ObjectType = "bill"
	// ObjectTypeSupplierCredit indicates that the object is a supplier credit: a supplier's credit note, read by the buyer.
	ObjectTypeSupplierCredit ObjectType = "supplier_credit"
	// ObjectTypeSupplierItem indicates that the object is an item as one supplier sells it.
	ObjectTypeSupplierItem ObjectType = "supplier_item"
	// ObjectTypeInventoryMovement indicates that the object is an inventory movement at one location.
	ObjectTypeInventoryMovement ObjectType = "inventory_movement"
	// ObjectTypeInventoryLevel indicates that the object is an item's inventory level at one location.
	ObjectTypeInventoryLevel ObjectType = "inventory_level"
	// ObjectTypeLotSizeDefault indicates that the object is a default lot size.
	ObjectTypeLotSizeDefault ObjectType = "lot_size_default"
	// ObjectTypeBatchScan indicates that the object is a recorded batch scan.
	ObjectTypeBatchScan ObjectType = "batch_scan"
	// ObjectTypeBatchScanPreview indicates that the object is what a batch scan would do, without saving it.
	ObjectTypeBatchScanPreview ObjectType = "batch_scan_preview"
	// ObjectTypeWorkCenter indicates that the object is a work center.
	ObjectTypeWorkCenter ObjectType = "work_center"
	// ObjectTypeDevice indicates that the object is an enrolled device, such as a shop-floor terminal.
	ObjectTypeDevice ObjectType = "device"
	// ObjectTypeDevicePairing indicates that the object is the pairing of a device to an account.
	ObjectTypeDevicePairing ObjectType = "device_pairing"
	// ObjectTypeNote indicates that the object is a note attributed to its author.
	ObjectTypeNote ObjectType = "note"
	// ObjectTypeSubscription indicates that the object is a subscription to notifications about a record.
	ObjectTypeSubscription ObjectType = "subscription"
	// ObjectTypeNotificationPreferences indicates that the object is a user's notification preferences.
	ObjectTypeNotificationPreferences ObjectType = "notification_preferences"
	// ObjectTypeNotificationDefaults indicates that the object is an account's default notification preferences.
	ObjectTypeNotificationDefaults ObjectType = "notification_defaults"
	// ObjectTypeAgent indicates that the object is an agent.
	ObjectTypeAgent ObjectType = "agent"
	// ObjectTypeAgentVersion indicates that the object is one version of an agent.
	ObjectTypeAgentVersion ObjectType = "agent_version"
	// ObjectTypeTool indicates that the object is a tool an agent can call.
	ObjectTypeTool ObjectType = "tool"
	// ObjectTypeToolCall indicates that the object is one call an agent made to a tool.
	ObjectTypeToolCall ObjectType = "tool_call"
	// ObjectTypeAgentSignUp indicates that the object is an agent's sign-up into a sandbox account.
	ObjectTypeAgentSignUp ObjectType = "agent_sign_up"
	// ObjectTypeRequestLogBodies indicates that the object is the request and response bodies of a request log.
	ObjectTypeRequestLogBodies ObjectType = "request_log_bodies"
	// ObjectTypeEmailLogRelated indicates that the object is the record an email log carried.
	ObjectTypeEmailLogRelated ObjectType = "email_log_related"
	// ObjectTypeSearchResult indicates that the object is one search result.
	ObjectTypeSearchResult ObjectType = "search_result"
	// ObjectTypeRoleTemplate indicates that the object is a role template, a starting point for a role.
	ObjectTypeRoleTemplate ObjectType = "role_template"
	// ObjectTypeDemandAdjustment indicates that the object is an adjustment to planned demand.
	ObjectTypeDemandAdjustment ObjectType = "demand_adjustment"
	// ObjectTypePlanningSettings indicates that the object is an account's planning settings.
	ObjectTypePlanningSettings ObjectType = "planning_settings"
)

func init() {
	object.Register(
		ObjectTypeJob, ObjectTypeWorkOrder, ObjectTypeManufacturingMethod, ObjectTypeMethodOperation, ObjectTypeMethodMaterial,
		ObjectTypePriceList, ObjectTypePriceListRule, ObjectTypeCreditNote, ObjectTypePayment, ObjectTypeApplication,
		ObjectTypeBill, ObjectTypeSupplierCredit, ObjectTypeSupplierItem, ObjectTypeInventoryMovement, ObjectTypeInventoryLevel,
		ObjectTypeLotSizeDefault, ObjectTypeBatchScan, ObjectTypeBatchScanPreview, ObjectTypeWorkCenter, ObjectTypeDevice,
		ObjectTypeDevicePairing, ObjectTypeNote, ObjectTypeSubscription, ObjectTypeNotificationPreferences,
		ObjectTypeNotificationDefaults, ObjectTypeAgent, ObjectTypeAgentVersion, ObjectTypeTool, ObjectTypeToolCall,
		ObjectTypeAgentSignUp, ObjectTypeRequestLogBodies, ObjectTypeSearchResult, ObjectTypeRoleTemplate,
		ObjectTypeDemandAdjustment, ObjectTypePlanningSettings, ObjectTypeEmailLogRelated,
	)
}
