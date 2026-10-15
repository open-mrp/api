package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeCustomerLeadTime,
		// A resolved lead time is never fetched by id — it is only ever the body of the endpoint that resolves it — but the registry requires a loader, so this one answers nothing. The two sub-fields do the real work.
		Load: loadCustomerLeadTimes,
		Subs: []include.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs: func(ctx context.Context, parent any) []string {
					if _, id, ok := customerLeadTimeSubID(ctx, parent, "customer_id"); ok {
						return []string{id}
					}
					return nil
				},
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					if lt, id, ok := customerLeadTimeSubID(ctx, parent, "customer_id"); ok {
						if customer, ok := loaded[id].(*apiresource.Customer); ok {
							lt.Customer = customer
						}
					}
				},
			},
			{
				Key:         "account_group",
				Target:      constants.ObjectTypeAccountGroup,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractAccountGroupIDFromCustomerLeadTime,
				Populate:    populateAccountGroupOnCustomerLeadTime,
			},
			{
				Key:         "parent_customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractParentCustomerIDFromCustomerLeadTime,
				Populate:    populateParentCustomerOnCustomerLeadTime,
			},
		},
	})
}

func loadCustomerLeadTimes(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// customerLeadTimeMetaID is the customer the lead time was resolved for, which is what the ids below are keyed by. A
// resolved lead time has no id of its own, and is only ever the body of one request, so it is the only one stashed.
func customerLeadTimeMetaID(lt *apiresource.CustomerLeadTime) (string, bool) {
	return customerLeadTimeKey, lt != nil
}

// customerLeadTimeKey keys the one lead time a response holds.
const customerLeadTimeKey = "lead_time"

func customerLeadTimeSubID(ctx context.Context, parent any, key string) (*apiresource.CustomerLeadTime, string, bool) {
	lt, ok := parent.(*apiresource.CustomerLeadTime)
	if !ok {
		return nil, "", false
	}
	metaID, ok := customerLeadTimeMetaID(lt)
	if !ok {
		return nil, "", false
	}
	id, ok := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomerLeadTime, metaID, key)
	if !ok || id == "" {
		return nil, "", false
	}
	return lt, id, true
}

func extractAccountGroupIDFromCustomerLeadTime(ctx context.Context, parent any) []string {
	_, id, ok := customerLeadTimeSubID(ctx, parent, "account_group_id")
	if !ok {
		return nil
	}
	return []string{id}
}

func populateAccountGroupOnCustomerLeadTime(ctx context.Context, parent any, loaded map[string]any) {
	lt, id, ok := customerLeadTimeSubID(ctx, parent, "account_group_id")
	if !ok {
		return
	}
	if group, ok := loaded[id].(*apiresource.AccountGroup); ok {
		lt.AccountGroup = group
	}
}

func extractParentCustomerIDFromCustomerLeadTime(ctx context.Context, parent any) []string {
	_, id, ok := customerLeadTimeSubID(ctx, parent, "parent_customer_id")
	if !ok {
		return nil
	}
	return []string{id}
}

func populateParentCustomerOnCustomerLeadTime(ctx context.Context, parent any, loaded map[string]any) {
	lt, id, ok := customerLeadTimeSubID(ctx, parent, "parent_customer_id")
	if !ok {
		return
	}
	if customer, ok := loaded[id].(*apiresource.Customer); ok {
		lt.ParentCustomer = customer
	}
}
