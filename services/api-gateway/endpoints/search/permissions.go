package searchep

import (
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
)

// searchReadPermissions is the any-of set for unified search: the gateway rejects callers who hold none of these, and the handler only queries resource types the caller can read.
var searchReadPermissions = types.AnyOfPermissions{
	{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead},
	{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionRead},
	{Domain: types.PermissionDomainInvoices, Action: types.ActionRead},
	{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
	{Domain: types.PermissionDomainItems, Action: types.ActionRead},
	{Domain: types.PermissionDomainShipments, Action: types.ActionRead},
	{Domain: types.PermissionDomainMessaging, Action: types.ActionRead},
	{Domain: types.PermissionDomainAgents, Action: types.ActionRead},
}

// searchObjectTypes are the resource types unified search can return.
var searchObjectTypes = map[constants.ObjectType]struct{}{
	constants.ObjectTypeSalesOrder:       {},
	constants.ObjectTypePurchaseOrder:    {},
	constants.ObjectTypeInvoice:          {},
	constants.ObjectTypeCustomer:         {},
	constants.ObjectTypeItem:             {},
	constants.ObjectTypeProduct:          {},
	constants.ObjectTypeShipment:         {},
	constants.ObjectTypeMessagingContact: {},
	constants.ObjectTypeAgentDefinition:  {},
}

func isSearchObjectType(objectType constants.ObjectType) bool {
	_, ok := searchObjectTypes[objectType]
	return ok
}
