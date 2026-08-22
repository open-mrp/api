package productionstepep

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

// Request to update a production output.
type UpdateProductionRequest struct {
	// Production step ID.
	ProductionStepID string `path:"production_step_id" validate:"required"`
	// Production ID.
	ProductionID string `path:"id" validate:"required"`
	// Item this step produces.
	//
	// Changing the item recomputes all of the step's connections in the production flow graph from the items it now produces and consumes, which discards connections that were made manually.
	ItemID field.Optional[string] `json:"item_id,omitzero" validate:"omitempty"`
	// Quantity value as a decimal string.
	//
	// Ignored unless `quantity_unit_id` is also provided.
	QuantityValue field.Optional[string] `json:"quantity_value,omitzero"`
	// Unit ID for `quantity_value`.
	//
	// Ignored unless `quantity_value` is also provided.
	QuantityUnitID field.Optional[string] `json:"quantity_unit_id,omitzero" validate:"omitempty"`
}

var sampleUpdateProductionItemID = apiresource.SampleItemID
var sampleUpdateProductionQuantityValue = "500"
var sampleUpdateProductionQuantityUnitID = apiresource.SampleUnitID
var sampleUpdateProductionRequest = &UpdateProductionRequest{
	ItemID:         field.Some(sampleUpdateProductionItemID),
	QuantityValue:  field.Some(sampleUpdateProductionQuantityValue),
	QuantityUnitID: field.Some(sampleUpdateProductionQuantityUnitID),
}

func (*UpdateProductionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductionRequest)
}

// Partially updates a production output within a production step.
type UpdateProductionEndpoint struct{}

func (e *UpdateProductionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductionRequest, *apiresource.ProductionOutput] {
	return (&apiendpoint.APIEndpoint[*UpdateProductionRequest, *apiresource.ProductionOutput]{
		Title:               "Update Production",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/operations/production-steps/{production_step_id}/productions/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionSteps, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeProduction,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductionRequest) (*apiresource.ProductionOutput, *apierror.APIError) {
			return svc.(ProductionStepSvc).UpdateProduction
		},
	})
}
