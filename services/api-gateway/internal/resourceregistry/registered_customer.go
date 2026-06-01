package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeCustomer,
		Load:       resourceloaders.LoadCustomers,
		Subs: []resourcekit.SubField{
			{Key: "credit_limit", Populate: populateCreditLimitOnCustomer},
			{Key: "contact_info", Populate: populateContactInfoOnCustomer},
			{Key: "freight_preferences", Populate: populateFreightPreferencesOnCustomer},
			{Key: "freight_preferences.carrier", Populate: populateCarrierOnCustomerFP},
			{Key: "freight_preferences.service_level", Populate: populateServiceLevelOnCustomerFP},
			{Key: "defaults", Populate: populateDefaultsOnCustomer},
			{Key: "defaults.payment_term", Populate: populatePaymentTermOnCustomerDefaults},
			{Key: "defaults.shipping_term", Populate: populateShippingTermOnCustomerDefaults},
			{Key: "defaults.priority", Populate: populatePriorityOnCustomerDefaults},
			{Key: "defaults.sales_rep", Populate: populateSalesRepOnCustomerDefaults},
			{Key: "notification_preferences", Populate: populateNotificationPreferencesOnCustomer},
			{Key: "bill_to_address", Populate: populateBillToAddressOnCustomer},
			{Key: "ship_to_address", Populate: populateShipToAddressOnCustomer},
			{Key: "type", Populate: populateTypeOnCustomer},
			{Key: "price_groups", Populate: populatePriceGroupsOnCustomer},
			{Key: "parent_account", Populate: populateParentAccountOnCustomer},
			{Key: "child_accounts", Populate: populateChildAccountsOnCustomer},
		},
	})
}

func populateCreditLimitOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "credit_limit")
	if !ok {
		return
	}
	c.CreditLimit = v.(*apiresource.Quantity)
}

func populateContactInfoOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "contact_info")
	if !ok {
		return
	}
	c.ContactInfo = v.(*apiresource.CustomerContactInfo)
}

func populateFreightPreferencesOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "freight_preferences")
	if !ok {
		return
	}
	c.FreightPreferences = v.(*apiresource.CustomerFreightPreferences)
}

func populateDefaultsOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "defaults")
	if !ok {
		return
	}
	c.Defaults = v.(*apiresource.CustomerDefaults)
}

func populateNotificationPreferencesOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "notification_preferences")
	if !ok {
		return
	}
	c.NotificationPreferences = v.(*apiresource.CustomerNotificationPreferences)
}

func populateBillToAddressOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "bill_to_address")
	if !ok {
		return
	}
	c.BillToAddress = v.(*apiresource.Address)
}

func populateShipToAddressOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "ship_to_address")
	if !ok {
		return
	}
	c.ShipToAddress = v.(*apiresource.Address)
}

func populateTypeOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "type")
	if !ok {
		return
	}
	c.Type = v.(*apiresource.AccountGroup)
}

func populatePriceGroupsOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "price_groups")
	if !ok {
		return
	}
	c.PriceGroups = v.(*apiresource.List[apiresource.AccountGroup])
}

func populateParentAccountOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "parent_account")
	if !ok {
		return
	}
	c.ParentAccount = v.(*apiresource.Customer)
}

func populateChildAccountsOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "child_accounts")
	if !ok {
		return
	}
	c.ChildAccounts = v.(*apiresource.List[apiresource.Customer])
}

func populateCarrierOnCustomerFP(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.FreightPreferences == nil {
		return
	}
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "fp_carrier")
	if !ok {
		return
	}
	c.FreightPreferences.Carrier = v.(*apiresource.Carrier)
}

func populateServiceLevelOnCustomerFP(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.FreightPreferences == nil {
		return
	}
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "fp_service_level")
	if !ok {
		return
	}
	c.FreightPreferences.ServiceLevel = v.(*apiresource.ServiceLevel)
}

func populatePaymentTermOnCustomerDefaults(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.Defaults == nil {
		return
	}
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "defaults_payment_term")
	if !ok {
		return
	}
	c.Defaults.PaymentTerm = v.(*apiresource.PaymentTerm)
}

func populateShippingTermOnCustomerDefaults(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.Defaults == nil {
		return
	}
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "defaults_shipping_term")
	if !ok {
		return
	}
	c.Defaults.ShippingTerm = v.(*apiresource.ShippingTerm)
}

func populatePriorityOnCustomerDefaults(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.Defaults == nil {
		return
	}
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "defaults_priority")
	if !ok {
		return
	}
	c.Defaults.Priority = v.(*apiresource.Priority)
}

func populateSalesRepOnCustomerDefaults(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.Defaults == nil {
		return
	}
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, "defaults_sales_rep")
	if !ok {
		return
	}
	c.Defaults.SalesRep = v.(*apiresource.AccountUser)
}
