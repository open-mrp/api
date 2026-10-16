package resourceloaders

import (
	"context"
	"strings"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var customerLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.customer")

// LoadCustomers fetches customers by ID via BatchGetCustomersByIDs and presents them in full: their sections are
// always populated, and their related records load through their own includes.
func LoadCustomers(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, customerLoaderTracer, "loader.customers.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetCustomersByIDsResponse, error) {
			return coreClient.BatchGetCustomersByIDs(ctx, &pb.BatchGetCustomersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Customers))
	for _, c := range resp.Customers {
		out[c.Id] = CustomerFromProto(ctx, c)
	}
	return out, nil
}

// HydrateCustomerEntities fills in the display Name (customer name) and Handle (customer number) on entity references that point to a customer, which the bare entity_type/entity_id pair cannot supply. The entities are mutated in place via one batched BatchGetCustomersByIDs call. Best-effort: nil entries, non-customer entities, unresolved/deleted ids, or a failed customer fetch leave the entity as a plain id/type reference.
func HydrateCustomerEntities(ctx context.Context, entities []*apiresource.Entity) {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, e := range entities {
		if e == nil || e.Type != constants.ObjectTypeCustomer {
			continue
		}
		if _, ok := seen[e.ID]; ok {
			continue
		}
		seen[e.ID] = struct{}{}
		ids = append(ids, e.ID)
	}
	if len(ids) == 0 {
		return
	}
	loaded, apiErr := LoadCustomers(ctx, ids)
	if apiErr != nil {
		return
	}
	for _, e := range entities {
		if e == nil || e.Type != constants.ObjectTypeCustomer {
			continue
		}
		c, ok := loaded[e.ID].(*apiresource.Customer)
		if !ok {
			continue
		}
		name := c.Name
		number := c.Number
		e.Name = &name
		e.Handle = &number
	}
}

// Load-meta keys the customer presenter stashes for the includes that resolve a customer's related records.
const (
	CustomerMetaGroupID               = "group_id"
	CustomerMetaPriceGroups           = "price_groups"
	CustomerMetaPriceListIDs          = "price_list_ids"
	CustomerMetaParentCustomerID      = "parent_customer_id"
	CustomerMetaChildCustomerIDs      = "child_customer_ids"
	CustomerMetaRecipients            = "notification_recipients"
	CustomerMetaRecipientUsers        = "notification_recipient_users"
	CustomerMetaCarrierID             = "defaults_carrier_id"
	CustomerMetaServiceLevelID        = "defaults_service_level_id"
	CustomerMetaPaymentTermID         = "defaults_payment_term_id"
	CustomerMetaShippingTermID        = "defaults_shipping_term_id"
	CustomerMetaSalesRepID            = "defaults_sales_rep_id"
	CustomerMetaBillToAddress         = "defaults_bill_to_address"
	CustomerMetaShipToAddress         = "defaults_ship_to_address"
	CustomerMetaReceiveCalendarID     = "defaults_receive_calendar_id"
	customerDayUnitID                 = "day"
	customerDefaultCarrierBillingType = constants.CarrierBillingTypeSender
)

// CustomerFromProto is a customer as the response shows it: its own fields and sections, with the IDs of its
// related records stashed for their includes. Related records stay null until included.
func CustomerFromProto(ctx context.Context, c *pb.CustomerProto) *apiresource.Customer {
	if c == nil {
		return nil
	}
	relationship := constants.CustomerRelationshipTypeStandalone
	if c.IsParentAccount {
		relationship = constants.CustomerRelationshipTypeParent
	} else if c.ParentAccount != nil {
		relationship = constants.CustomerRelationshipTypeChild
	}
	status := constants.ArchiveStatus(c.ArchiveStatus)
	if !status.IsValid() {
		status = constants.ArchiveStatusActive
	}
	billing := customerDefaultCarrierBillingType
	if c.CarrierBillingType != nil && *c.CarrierBillingType != "" {
		billing = constants.CarrierBillingType(*c.CarrierBillingType)
	}
	cust := &apiresource.Customer{
		ID:               c.Id,
		Object:           constants.ObjectTypeCustomer,
		Status:           status,
		Standing:         constants.CustomerStandingFromStatusCode(c.Status),
		Name:             c.Name,
		Number:           c.Number,
		RelationshipType: relationship,
		Instructions:     c.Instructions,
		CommissionPolicy: new(constants.CommissionPolicy(c.CommissionPolicy)),
		FreightPolicy:    constants.FreightPolicy(c.FreightPolicy),
		Contact:          &apiresource.CustomerContact{Email: c.Email, Phone: c.Phone, URL: c.Url},
		CarrierBilling:   &apiresource.CustomerCarrierBilling{Type: billing, AccountNumber: c.CarrierBillingAccount},
		Defaults:         &apiresource.CustomerDefaults{},
		CreatedAt:        grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(c.UpdatedAt),
	}
	if c.CreditLimit != nil {
		cust.CreditLimit = MoneyFromProto(c.CreditLimit)
	}
	cust.Defaults.LeadTime = DaysQuantity(c.DefaultLeadTimeDays)
	if c.Supply != nil && *c.Supply != "" {
		supply := constants.Supply(*c.Supply)
		cust.Defaults.Supply = &supply
	}
	stashCustomerRefs(ctx, cust.ID, c)
	return cust
}

// stashCustomerRefs records the IDs (and the records the proto already holds in full) that the customer's
// includes resolve.
func stashCustomerRefs(ctx context.Context, id string, c *pb.CustomerProto) {
	meta := include.GetLoadMeta(ctx)
	set := func(key, value string) {
		if value != "" {
			meta.Set(constants.ObjectTypeCustomer, id, key, value)
		}
	}
	if c.TypeGroup != nil {
		set(CustomerMetaGroupID, c.TypeGroup.Id)
	}
	if c.ParentAccount != nil {
		set(CustomerMetaParentCustomerID, c.ParentAccount.Id)
	}
	if c.DefaultCarrier != nil {
		set(CustomerMetaCarrierID, c.DefaultCarrier.Id)
	}
	if c.DefaultServiceLevel != nil {
		set(CustomerMetaServiceLevelID, c.DefaultServiceLevel.Id)
	}
	if c.DefaultPaymentTerm != nil {
		set(CustomerMetaPaymentTermID, c.DefaultPaymentTerm.Id)
	}
	if c.DefaultShippingTerm != nil {
		set(CustomerMetaShippingTermID, c.DefaultShippingTerm.Id)
	}
	if c.DefaultSalesRep != nil {
		set(CustomerMetaSalesRepID, c.DefaultSalesRep.Id)
	}
	if c.ReceiveCalendarId != nil {
		set(CustomerMetaReceiveCalendarID, *c.ReceiveCalendarId)
	}
	if c.BillToAddress != nil {
		meta.Set(constants.ObjectTypeCustomer, id, CustomerMetaBillToAddress, CustomerAddressFromProto(c.BillToAddress))
	}
	if c.ShipToAddress != nil {
		meta.Set(constants.ObjectTypeCustomer, id, CustomerMetaShipToAddress, CustomerAddressFromProto(c.ShipToAddress))
	}
	if len(c.ChildAccounts) > 0 {
		childIDs := make([]string, 0, len(c.ChildAccounts))
		for _, child := range c.ChildAccounts {
			if child.Id != "" {
				childIDs = append(childIDs, child.Id)
			}
		}
		meta.Set(constants.ObjectTypeCustomer, id, CustomerMetaChildCustomerIDs, childIDs)
	}
	priceGroups := make([]apiresource.AccountGroup, len(c.PriceGroups))
	for i, g := range c.PriceGroups {
		priceGroups[i] = apiresource.AccountGroup{
			ID:               g.Id,
			Object:           constants.ObjectTypeAccountGroup,
			Name:             g.Name,
			CommissionPolicy: new(constants.CommissionPolicy(g.CommissionPolicy)),
			FreightPolicy:    constants.FreightPolicy(g.FreightPolicy),
			Type:             constants.AccountGroupType(g.Type),
			Defaults:         &apiresource.AccountGroupDefaults{},
			CreatedAt:        grpcutil.TimestampToTime(g.CreatedAt),
			UpdatedAt:        grpcutil.TimestampToTime(g.UpdatedAt),
		}
	}
	meta.Set(constants.ObjectTypeCustomer, id, CustomerMetaPriceGroups, apiresource.NewList(priceGroups, apiresource.PageInfo{}))
	meta.Set(constants.ObjectTypeCustomer, id, CustomerMetaPriceListIDs, c.PriceListIds)
	if c.HasNotificationRecipients {
		recipients := make([]apiresource.EmailRecipient, len(c.NotificationRecipients))
		users := make(map[string]*apiresource.AccountUser, len(c.NotificationRecipients))
		for i, r := range c.NotificationRecipients {
			docs := make([]constants.RecipientDocument, len(r.Documents))
			for j, d := range r.Documents {
				docs[j] = constants.RecipientDocument(d)
			}
			recipients[i] = apiresource.EmailRecipient{Documents: docs}
			if r.AccountUser != nil {
				users[r.AccountUserId] = AccountUserFromDetail(ctx, r.AccountUser)
			}
		}
		recipientUserIDs := make([]string, len(c.NotificationRecipients))
		for i, r := range c.NotificationRecipients {
			recipientUserIDs[i] = r.AccountUserId
		}
		meta.Set(constants.ObjectTypeCustomer, id, CustomerMetaRecipients, apiresource.NewList(recipients, apiresource.PageInfo{}))
		meta.Set(constants.ObjectTypeCustomer, id, CustomerMetaRecipientUsers, customerRecipientUsers{ids: recipientUserIDs, byID: users})
	}
}

// customerRecipientUsers are a customer's recipients' account users, in recipient order.
type customerRecipientUsers struct {
	ids  []string
	byID map[string]*apiresource.AccountUser
}

// CustomerRecipientUser is the account user of a customer's i-th notification recipient, when they were loaded.
func CustomerRecipientUser(ctx context.Context, customerID string, i int) *apiresource.AccountUser {
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeCustomer, customerID, CustomerMetaRecipientUsers)
	if !ok {
		return nil
	}
	users := v.(customerRecipientUsers)
	if i < 0 || i >= len(users.ids) {
		return nil
	}
	return users.byID[users.ids[i]]
}

// CustomerCoreIncludes are the includes core-service needs to load what a customer request's includes ask for.
func CustomerCoreIncludes(ctx context.Context) []string {
	var out []string
	seen := map[string]bool{}
	for _, key := range include.RequestedIncludes(ctx) {
		for prefix, core := range customerCoreIncludeKeys {
			if (key == prefix || strings.HasPrefix(key, prefix+".")) && !seen[core] {
				seen[core] = true
				out = append(out, core)
			}
		}
	}
	return out
}

var customerCoreIncludeKeys = map[string]string{
	"defaults.bill_to_address": "bill_to_address",
	"defaults.ship_to_address": "ship_to_address",
	"child_customers":          "child_accounts",
	"price_groups":             "price_groups",
	"notification_recipients":  "notification_recipients",
}

// CustomerAddressFromProto is a customer's or supplier's default address as the response shows it. nil stays nil.
func CustomerAddressFromProto(a *pb.CustomerAddressProto) *apiresource.Address {
	if a == nil {
		return nil
	}
	parts := AddressParts{
		ID:         a.Id,
		Name:       a.Name,
		IsDropShip: a.IsDropShip,
		Phone:      a.Phone,
		Email:      a.Email,
		CreatedAt:  grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(a.UpdatedAt),
	}
	if g := a.Geolocation; g != nil {
		parts.Line1, parts.Line2, parts.City, parts.State, parts.PostalCode, parts.Country = g.StreetLine_1, g.StreetLine_2, g.Locality, g.State, g.PostalCode, g.Country
	}
	return NewAddress(parts)
}
