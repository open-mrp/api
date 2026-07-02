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
			// credit_limit is a Quantity built inline from the customer proto; declaring its Target lets the
			// resolver recurse into credit_limit.unit (the Quantity's unit_id is stashed under ObjectTypeQuantity).
			{Key: "credit_limit", Target: constants.ObjectTypeQuantity, ExtractRefs: extractCreditLimitRefFromCustomer, Populate: populateCreditLimitOnCustomer},
			{Key: "contact_info", Populate: populateContactInfoOnCustomer},
			{Key: "freight_preferences", Populate: populateFreightPreferencesOnCustomer},
			// freight_preferences.carrier is fetched by id via LoadCarriers (not built inline) so the loaded
			// Carrier carries its service_level_ids preview, enabling the nested freight_preferences.carrier.service_levels include.
			{Key: "freight_preferences.carrier", Target: constants.ObjectTypeCarrier, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractFPCarrierIDFromCustomer, Populate: populateCarrierOnCustomerFP},
			{Key: "freight_preferences.service_level", Populate: populateServiceLevelOnCustomerFP},
			{Key: "defaults", Populate: populateDefaultsOnCustomer},
			{Key: "defaults.payment_term", Populate: populatePaymentTermOnCustomerDefaults},
			{Key: "defaults.shipping_term", Populate: populateShippingTermOnCustomerDefaults},
			{Key: "defaults.priority", Populate: populatePriorityOnCustomerDefaults},
			{
				Key:         "defaults.sales_rep",
				Target:      constants.ObjectTypeAccountUser,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractSalesRepIDFromCustomerDefaults,
				Populate:    populateSalesRepOnCustomerDefaults,
			},
			{Key: "notification_preferences", Populate: populateNotificationPreferencesOnCustomer},
			{Key: "bill_to_address", Populate: populateBillToAddressOnCustomer},
			{Key: "ship_to_address", Populate: populateShipToAddressOnCustomer},
			{Key: "type", Populate: populateTypeOnCustomer},
			{Key: "price_groups", Populate: populatePriceGroupsOnCustomer},
			{Key: "parent_account", Target: constants.ObjectTypeCustomer, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractParentAccountIDFromCustomer, Populate: populateParentAccountOnCustomer},
			{Key: "child_accounts", Target: constants.ObjectTypeCustomer, Cardinality: resourcekit.CardinalityList, ExtractIDs: extractChildAccountIDsFromCustomer, Populate: populateChildAccountsOnCustomer},
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

// extractCreditLimitRefFromCustomer returns the inline credit_limit Quantity (attached by populateCreditLimitOnCustomer) so the resolver can recurse into credit_limit.unit.
func extractCreditLimitRefFromCustomer(_ context.Context, parent any) []any {
	c := parent.(*apiresource.Customer)
	if c.CreditLimit == nil {
		return nil
	}
	return []any{c.CreditLimit}
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

func extractParentAccountIDFromCustomer(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.Customer)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomer, c.ID, "parent_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateParentAccountOnCustomer(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.Customer)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomer, c.ID, "parent_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		c.ParentAccount = v.(*apiresource.Customer)
	}
}

func extractChildAccountIDsFromCustomer(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.Customer)
	ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeCustomer, c.ID, "child_account_ids")
	return ids
}

func populateChildAccountsOnCustomer(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.Customer)
	ids, _ := resourcekit.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeCustomer, c.ID, "child_account_ids")
	items := make([]apiresource.Customer, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.Customer)))
		}
	}
	c.ChildAccounts = apiresource.NewList(items, apiresource.PageInfo{})
}

// extractFPCarrierIDFromCustomer returns the customer's default-carrier id (stashed by stashCustomerMeta) so the resolver fetches the full Carrier via LoadCarriers — which carries the service_level_ids preview the nested freight_preferences.carrier.service_levels include needs.
func extractFPCarrierIDFromCustomer(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.Customer)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomer, c.ID, "fp_carrier_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCarrierOnCustomerFP(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.FreightPreferences == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomer, c.ID, "fp_carrier_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		c.FreightPreferences.Carrier = v.(*apiresource.Carrier)
	}
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

func extractSalesRepIDFromCustomerDefaults(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.Customer)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeCustomer, c.ID, "defaults_sales_rep_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateSalesRepOnCustomerDefaults(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.Defaults == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeCustomer, c.ID, "defaults_sales_rep_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		c.Defaults.SalesRep = v.(*apiresource.AccountUser)
	}
}
