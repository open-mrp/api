package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeCustomer,
		Load:       resourceloaders.LoadCustomers,
		Subs: []include.SubField{
			customerRef("group", constants.ObjectTypeAccountGroup, resourceloaders.CustomerMetaGroupID, func(c *apiresource.Customer, v any) {
				c.Group = v.(*apiresource.AccountGroup)
			}),
			{Key: "price_groups", Populate: populatePriceGroupsOnCustomer},
			{
				Key:         "price_lists",
				Target:      constants.ObjectTypePriceList,
				Cardinality: include.CardinalityList,
				ExtractIDs:  metaIDs(constants.ObjectTypeCustomer, func(p any) string { return p.(*apiresource.Customer).ID }, resourceloaders.CustomerMetaPriceListIDs),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					c := parent.(*apiresource.Customer)
					ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeCustomer, c.ID, resourceloaders.CustomerMetaPriceListIDs)
					c.PriceLists = apiresource.NewList(loadedInOrder[apiresource.PriceList](ids, loaded), apiresource.PageInfo{})
				},
			},
			customerRef("parent_customer", constants.ObjectTypeCustomer, resourceloaders.CustomerMetaParentCustomerID, func(c *apiresource.Customer, v any) {
				c.ParentCustomer = v.(*apiresource.Customer)
			}),
			{Key: "child_customers", Target: constants.ObjectTypeCustomer, Cardinality: include.CardinalityList, ExtractIDs: extractChildCustomerIDs, Populate: populateChildCustomers},
			{Key: "notification_recipients", Populate: populateNotificationRecipientsOnCustomer},
			{Key: "notification_recipients.account_user", Populate: populateRecipientAccountUsersOnCustomer},
			customerRef("defaults.carrier", constants.ObjectTypeCarrier, resourceloaders.CustomerMetaCarrierID, func(c *apiresource.Customer, v any) {
				c.Defaults.Carrier = v.(*apiresource.Carrier)
			}),
			customerRef("defaults.service_level", constants.ObjectTypeServiceLevel, resourceloaders.CustomerMetaServiceLevelID, func(c *apiresource.Customer, v any) {
				c.Defaults.ServiceLevel = v.(*apiresource.ServiceLevel)
			}),
			customerRef("defaults.payment_term", constants.ObjectTypePaymentTerm, resourceloaders.CustomerMetaPaymentTermID, func(c *apiresource.Customer, v any) {
				c.Defaults.PaymentTerm = v.(*apiresource.PaymentTerm)
			}),
			customerRef("defaults.shipping_term", constants.ObjectTypeShippingTerm, resourceloaders.CustomerMetaShippingTermID, func(c *apiresource.Customer, v any) {
				c.Defaults.ShippingTerm = v.(*apiresource.ShippingTerm)
			}),
			customerRef("defaults.sales_rep", constants.ObjectTypeAccountUser, resourceloaders.CustomerMetaSalesRepID, func(c *apiresource.Customer, v any) {
				c.Defaults.SalesRep = v.(*apiresource.AccountUser)
			}),
			customerRef("defaults.receive_calendar", constants.ObjectTypeOperatingCalendar, resourceloaders.CustomerMetaReceiveCalendarID, func(c *apiresource.Customer, v any) {
				c.Defaults.ReceiveCalendar = v.(*apiresource.OperatingCalendar)
			}),
			{Key: "defaults.bill_to_address", Populate: func(ctx context.Context, parent any, _ map[string]any) {
				c := parent.(*apiresource.Customer)
				if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, resourceloaders.CustomerMetaBillToAddress); ok && c.Defaults != nil {
					c.Defaults.BillToAddress = v.(*apiresource.Address)
				}
			}},
			{Key: "defaults.ship_to_address", Populate: func(ctx context.Context, parent any, _ map[string]any) {
				c := parent.(*apiresource.Customer)
				if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, resourceloaders.CustomerMetaShipToAddress); ok && c.Defaults != nil {
					c.Defaults.ShipToAddress = v.(*apiresource.Address)
				}
			}},
		},
	})
}

// customerRef declares an include that loads one record a customer names by the ID its presenter stashed under
// metaKey, and sets it with set.
func customerRef(key string, target constants.ObjectType, metaKey string, set func(c *apiresource.Customer, v any)) include.SubField {
	idOf := func(ctx context.Context, parent any) string {
		id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomer, parent.(*apiresource.Customer).ID, metaKey)
		return id
	}
	return include.SubField{
		Key:         key,
		Target:      target,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			if id := idOf(ctx, parent); id != "" {
				return []string{id}
			}
			return nil
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			c := parent.(*apiresource.Customer)
			if c.Defaults == nil {
				c.Defaults = &apiresource.CustomerDefaults{}
			}
			if v, ok := loaded[idOf(ctx, parent)]; ok && v != nil {
				set(c, v)
			}
		},
	}
}

func populatePriceGroupsOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, resourceloaders.CustomerMetaPriceGroups); ok {
		c.PriceGroups = v.(*apiresource.List[apiresource.AccountGroup])
	}
}

func extractChildCustomerIDs(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.Customer)
	ids, _ := include.GetLoadMeta(ctx).GetStrings(constants.ObjectTypeCustomer, c.ID, resourceloaders.CustomerMetaChildCustomerIDs)
	return ids
}

func populateChildCustomers(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.Customer)
	ids := extractChildCustomerIDs(ctx, parent)
	items := make([]apiresource.Customer, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id].(*apiresource.Customer); ok {
			items = append(items, *v)
		}
	}
	c.ChildCustomers = apiresource.NewList(items, apiresource.PageInfo{})
}

func populateNotificationRecipientsOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, c.ID, resourceloaders.CustomerMetaRecipients); ok {
		c.NotificationRecipients = v.(*apiresource.List[apiresource.EmailRecipient])
	}
}

// populateRecipientAccountUsersOnCustomer sets each recipient's account user, which core-service loaded with the
// recipients: they belong to the customer's account, which the account-user loader can't reach.
func populateRecipientAccountUsersOnCustomer(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Customer)
	if c.NotificationRecipients == nil {
		populateNotificationRecipientsOnCustomer(ctx, parent, nil)
	}
	if c.NotificationRecipients == nil {
		return
	}
	for i := range c.NotificationRecipients.Data {
		c.NotificationRecipients.Data[i].AccountUser = resourceloaders.CustomerRecipientUser(ctx, c.ID, i)
	}
}
