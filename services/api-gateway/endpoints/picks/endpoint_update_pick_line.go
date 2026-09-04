package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a pick line's picked quantity.
type UpdatePickLineRequest struct {
	// Pick ID.
	PickID string `path:"pick_id" validate:"required"`
	// Pick line ID.
	PickLineID string `path:"id" validate:"required"`
	// New picked quantity for the line, as a decimal string read in the unit the sales order line was sold in, stored as given and not capped at the ordered quantity.
	//
	// Must not be negative. Pulling more than was ordered is a real floor event and is kept as recorded; pulling a negative amount is not.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero" validate:"omitempty,decimal,gte=0" format:"decimal"`
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
		Public:            true,
		Preview:           true,
		AgentTool:         true,
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
				"quantity.unit",
				"ordered_quantity.unit",
			},
		}),
	})
}
