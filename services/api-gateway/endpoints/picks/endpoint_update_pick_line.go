package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdatePickLineRequest is the request to update a pick line's quantity.
type UpdatePickLineRequest struct {
	// Pick ID.
	PickID string `path:"pick_id" validate:"required"`
	// Pick line ID.
	PickLineID string `path:"id" validate:"required"`
	// Quantity value to set for this line.
	QuantityValue *string `json:"quantity_value,omitempty" nullable:"false"`
}

var sampleUpdatePickLineQuantityValue = "10.000000000000000000000000000000"
var sampleUpdatePickLineRequest = &UpdatePickLineRequest{
	QuantityValue: &sampleUpdatePickLineQuantityValue,
}

func (*UpdatePickLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePickLineRequest)
}

// Partially updates a pick line's quantity value.
type UpdatePickLineEndpoint struct{}

func (e *UpdatePickLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePickLineRequest, *apiresource.PickLineDetail] {
	return (&apiendpoint.APIEndpoint[*UpdatePickLineRequest, *apiresource.PickLineDetail]{
		Title:             "Update Pick Line",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/picks/{pick_id}/lines/{id}",
		ContentType:       "application/json",
		Request:           &UpdatePickLineRequest{},
		Response:          &apiresource.PickLineDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
			return svc.(PickSvc).UpdatePickLine
		},
	}).WithDocSource(e)
}
