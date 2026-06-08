package partep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a part.
type UpdatePartRequest struct {
	// Part ID.
	ItemID string `path:"id" validate:"required"`
	// SKU.
	SKU field.Optional[string] `json:"sku,omitzero" validate:"omitempty,max=255"`
	// Description.
	Description field.Clearable[string] `json:"description,omitzero"`
	// Notes.
	Notes field.Clearable[string] `json:"notes,omitzero"`
}

var sampleUpdatePartSKU = apiresource.SamplePartSKU
var sampleUpdatePartDescription = "Deep groove ball bearing, 20x47x14mm"
var sampleUpdatePartRequest = &UpdatePartRequest{
	SKU:         field.SomePtr(&sampleUpdatePartSKU),
	Description: field.Set(sampleUpdatePartDescription),
}

func (*UpdatePartRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePartRequest)
}

// Partially updates a part. Fields not provided retain their current values.
type UpdatePartEndpoint struct{}

func (e *UpdatePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePartRequest, *apiresource.Part] {
	return (&apiendpoint.APIEndpoint[*UpdatePartRequest, *apiresource.Part]{
		Title:             "Update Part",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/parts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).UpdatePart
		},
		ObjectType: constants.ObjectTypePart,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
