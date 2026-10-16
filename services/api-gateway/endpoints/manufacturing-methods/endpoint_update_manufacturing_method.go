package manufacturingmethodep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/field"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to update a manufacturing method.
type UpdateManufacturingMethodRequest struct {
	// Manufacturing method ID.
	ID string `path:"id" validate:"required"`
	// How much of the item one run of the method makes.
	OutputQuantity field.Optional[apirequest.QuantityInput] `json:"output_quantity,omitzero"`
}

var sampleUpdateManufacturingMethodRequest = &UpdateManufacturingMethodRequest{
	OutputQuantity: field.Some(apirequest.QuantityInput{Value: "24", UnitID: apiresource.SampleUnitID}),
}

func (*UpdateManufacturingMethodRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateManufacturingMethodRequest)
}

// Updates a draft manufacturing method.
//
// Only the fields you send change. Active and archived methods can't change (409 `invalid_state_transition`); create a new version to edit one.
type UpdateManufacturingMethodEndpoint struct{}

func (e *UpdateManufacturingMethodEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateManufacturingMethodRequest, *apiresource.ManufacturingMethod] {
	return &apiendpoint.APIEndpoint[*UpdateManufacturingMethodRequest, *apiresource.ManufacturingMethod]{
		Title:             "Update Manufacturing Method",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             methodsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeManufacturingMethod,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).UpdateManufacturingMethod
		},
		IncludeConfig: methodIncludes(),
	}
}
