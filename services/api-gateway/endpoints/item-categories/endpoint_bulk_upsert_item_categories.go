package itemcategoryep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// UpsertItemCategoryInput is the input for a single item category in a bulk upsert operation.
type UpsertItemCategoryInput struct {
	// Display name of the item category, used to match existing categories.
	Name string `json:"name" validate:"required,max=255"`
	// Item category type code. Create-only.
	Type constants.ItemCategoryType `json:"type" validate:"required"`
	// Unit group to associate with this item category, referenced by `id` or `name`.
	UnitGroup apirequest.ObjectIdentifier `json:"unit_group" validate:"required"`
	// Optional list of property names to attach to this category. Properties are matched
	// by name (case-insensitive) within the account; names not found are created automatically.
	// Relations are additive — existing relations are not removed.
	PropertyNames []string `json:"property_names" default:"[]"`
	// Optional notes.
	Notes *string `json:"notes" default:"null" nullable:"true"`
}

// BulkUpsertItemCategoriesRequest is the request to bulk upsert item categories.
type BulkUpsertItemCategoriesRequest struct {
	// Item categories to create or update, matched by name within the account.
	ItemCategories []UpsertItemCategoryInput `json:"item_categories" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertItemCategoriesRequest = &BulkUpsertItemCategoriesRequest{
	ItemCategories: []UpsertItemCategoryInput{
		{
			Name:      apiresource.SampleItemCategoryName,
			Type:      constants.ItemCategoryTypeMaterial,
			UnitGroup: apirequest.ObjectIdentifier{ID: apiresource.SampleUnitGroupID},
		},
	},
}

func (*BulkUpsertItemCategoriesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertItemCategoriesRequest)
}

// Creates or updates multiple item categories for the account, matched by name
// (case-insensitive), then writes asynchronously — 202 with a job to poll.
type BulkUpsertItemCategoriesEndpoint struct{}

func (e *BulkUpsertItemCategoriesEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertItemCategoriesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertItemCategoriesRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Item Categories",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/item-categories/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertItemCategoriesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(ItemCategorySvc).BulkUpsertItemCategories
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
