package itemcategoryep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// UpsertItemCategoryInput is the input for a single item category in a bulk upsert operation.
type UpsertItemCategoryInput struct {
	// Display name, such as `Fasteners`. Matches an existing category by name (case-insensitive).
	Name string `json:"name" validate:"required,max=255"`
	// What kind of items the category holds: `material` or `product`. Only used when the row creates a category.
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// Properties to attach, by name. Properties that don't exist are created. Properties already attached are never removed.
	PropertyNames []string `json:"property_names,omitzero"`
	// Unit group the category's items are measured in, by `id` or `name`. Defaults to each.
	UnitGroup field.Optional[apirequest.ObjectIdentifier] `json:"unit_group,omitzero" visibility:"internal"`
}

// BulkUpsertItemCategoriesRequest is the request to bulk upsert item categories.
type BulkUpsertItemCategoriesRequest struct {
	// Item categories to create or update, matched by name within the account.
	ItemCategories []UpsertItemCategoryInput `json:"item_categories" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertItemCategoriesRequest = &BulkUpsertItemCategoriesRequest{
	ItemCategories: []UpsertItemCategoryInput{
		{
			Name:          apiresource.SampleItemCategoryName,
			Type:          constants.ItemCategoryTypeMaterial,
			PropertyNames: []string{apiresource.SamplePropertyName},
		},
	},
}

func (*BulkUpsertItemCategoriesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertItemCategoriesRequest)
}

// Creates or updates item categories in bulk.
//
// Rows are matched to existing categories by `name` (case-insensitive). `property_names` attaches properties by name, creating any that don't exist, and never removes a property already attached. The work runs in the background; the response is an async job to poll.
type BulkUpsertItemCategoriesEndpoint struct{}

func (e *BulkUpsertItemCategoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertItemCategoriesRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertItemCategoriesRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Item Categories",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/item-categories/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCategories, Action: types.ActionCreate}, {Domain: types.PermissionDomainCategories, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertItemCategoriesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(ItemCategorySvc).BulkUpsertItemCategories
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
