package resourceregistry

import (
	"context"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeCustomerLeadTime,
		// A resolved lead time is never fetched by id — it is only ever the body of the endpoint that resolves it — but the registry requires a loader, so this one answers nothing. The two sub-fields do the real work.
		Load: loadCustomerLeadTimes,
		Subs: []resourcekit.SubField{
			{
				Key:         "account_group",
				Target:      constants.ObjectTypeAccountGroup,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractAccountGroupIDFromCustomerLeadTime,
				Populate:    populateAccountGroupOnCustomerLeadTime,
			},
			{
				Key:         "parent_customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractParentCustomerIDFromCustomerLeadTime,
				Populate:    populateParentCustomerOnCustomerLeadTime,
			},
		},
	})
}

func loadCustomerLeadTimes(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// customerLeadTimeMetaID is the customer the lead time was resolved for, which is what the ids below are keyed by. A resolved lead time has no id of its own.
func customerLeadTimeMetaID(lt *apiresource.CustomerLeadTime) (string, bool) {
	if lt == nil || lt.Customer == nil {
		return "", false
	}
	return lt.Customer.ID, true
}

func customerLeadTimeSubID(ctx context.Context, parent any, key string) (*apiresource.CustomerLeadTime, string, bool) {
	lt, ok := parent.(*apiresource.CustomerLeadTime)
	if !ok {
		return nil, "", false
	}
	metaID, ok := customerLeadTimeMetaID(lt)
	if !ok {
		return nil, "", false
	}
	id, ok := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeCustomerLeadTime, metaID, key)
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
