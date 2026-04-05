package partep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdatePartRequest is the request to partially update a part.
type UpdatePartRequest struct {
	// The item ID of the part to update.
	ItemID string `path:"id" validate:"required"`
	// The part SKU.
	SKU *string `json:"sku,omitempty"`
	// The part description.
	Description *string `json:"description"`
	// Optional notes about the part.
	Notes *string `json:"notes"`
}

var sampleUpdatePartSKU = apiresource.SamplePartSKU
var sampleUpdatePartDescription = "Deep groove ball bearing, 20x47x14mm"
var sampleUpdatePartRequest = &UpdatePartRequest{
	SKU:         &sampleUpdatePartSKU,
	Description: &sampleUpdatePartDescription,
}

func (*UpdatePartRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePartRequest)
}

type UpdatePartEndpoint struct{}

func (e *UpdatePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePartRequest, *apiresource.Part] {
	return &apiendpoint.APIEndpoint[*UpdatePartRequest, *apiresource.Part]{
		Title:             "Update Part",
		Description:       "Partially updates a part. Fields not provided retain their current values.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/parts/{id}",
		ContentType:       "application/json",
		Request:           &UpdatePartRequest{},
		Response:          &apiresource.Part{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).UpdatePart
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate"},
		}),
	}
}
