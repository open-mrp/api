package itemep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateItemRequest is the request to partially update an item.
type UpdateItemRequest struct {
	// The ID of the item to update.
	ItemID string `path:"id" validate:"required"`
	// The item SKU.
	SKU *string `json:"sku,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// The item description.
	Description *string `json:"description" nullable:"true"`
	// Optional notes about the item.
	Notes *string `json:"notes" nullable:"true"`
}

var sampleUpdateItemSKU = apiresource.SampleItemSKU
var sampleUpdateItemDescription = "Almond Butter, 16oz Jar"
var sampleUpdateItemRequest = &UpdateItemRequest{
	SKU:         &sampleUpdateItemSKU,
	Description: &sampleUpdateItemDescription,
}

func (*UpdateItemRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateItemRequest)
}

type UpdateItemEndpoint struct{}

func (e *UpdateItemEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateItemRequest, *apiresource.Item] {
	return &apiendpoint.APIEndpoint[*UpdateItemRequest, *apiresource.Item]{
		Title:             "Update Item",
		Description:       "Partially updates an item. Only provided fields are updated; omitted fields retain their current values.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/items/{id}",
		ContentType:       "application/json",
		Request:           &UpdateItemRequest{},
		Response:          &apiresource.Item{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateItemRequest) (*apiresource.Item, *apierror.APIError) {
			return svc.(ItemSvc).UpdateItem
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate", "attributes"},
		}),
	}
}
