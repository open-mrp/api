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

// Request to create a manufacturing method.
type CreateManufacturingMethodRequest struct {
	// ID of the item the method makes.
	ItemID string `json:"item_id" validate:"required"`
	// How much of the item one run of the method makes. Material quantities and labor and machine times are per this much output. Defaults to one of the item's stocking unit.
	OutputQuantity field.Optional[apirequest.QuantityInput] `json:"output_quantity,omitzero"`
}

var sampleCreateManufacturingMethodRequest = &CreateManufacturingMethodRequest{
	ItemID:         apiresource.SampleItemID,
	OutputQuantity: field.Some(apirequest.QuantityInput{Value: "12", UnitID: apiresource.SampleUnitID}),
}

func (*CreateManufacturingMethodRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateManufacturingMethodRequest)
}

// Creates a draft manufacturing method for an item.
//
// The draft takes the item's next version number. Add its operations and materials, then activate it to make it the method for new production.
type CreateManufacturingMethodEndpoint struct{}

func (e *CreateManufacturingMethodEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateManufacturingMethodRequest, *apiresource.ManufacturingMethod] {
	return &apiendpoint.APIEndpoint[*CreateManufacturingMethodRequest, *apiresource.ManufacturingMethod]{
		Title:             "Create Manufacturing Method",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             methodsRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeManufacturingMethod,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateManufacturingMethodRequest) (*apiresource.ManufacturingMethod, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).CreateManufacturingMethod
		},
		IncludeConfig: methodIncludes(),
		LocationFunc: func(resp *apiresource.ManufacturingMethod) string {
			return methodsRoute + "/" + resp.ID
		},
	}
}
