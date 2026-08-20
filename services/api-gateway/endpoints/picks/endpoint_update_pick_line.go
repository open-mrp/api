package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a pick line's picked quantity.
type UpdatePickLineRequest struct {
	// Pick ID.
	PickID string `path:"pick_id" validate:"required"`
	// Pick line ID.
	PickLineID string `path:"id" validate:"required"`
	// New picked quantity for the line, as a decimal string read in the unit the sales order line was sold in, stored as given and not capped at the ordered quantity.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero"`
}

var sampleUpdatePickLineQuantityValue = "10.000000000000000000000000000000"
var sampleUpdatePickLineRequest = &UpdatePickLineRequest{
	QuantityValue: field.Some(sampleUpdatePickLineQuantityValue),
}

func (*UpdatePickLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePickLineRequest)
}

// Updates a pick line's picked quantity.
//
// Use this to record a short or partial pick; Pick Pick Line fills in the full outstanding quantity instead.
type UpdatePickLineEndpoint struct{}

func (e *UpdatePickLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePickLineRequest, *apiresource.PickLine] {
	return (&apiendpoint.APIEndpoint[*UpdatePickLineRequest, *apiresource.PickLine]{
		Title:             "Update Pick Line",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/picks/{pick_id}/lines/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePickLine,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLine, *apierror.APIError) {
			return svc.(PickSvc).UpdatePickLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePickLine,
			Fields: []string{
				"sales_order_line",
				"sales_order_line.product",
				"quantity",
				"quantity.unit",
				"ordered_quantity",
				"ordered_quantity.unit",
			},
		}),
	})
}
