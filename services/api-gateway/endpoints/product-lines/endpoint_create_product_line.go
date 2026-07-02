package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a product line.
type CreateProductLineRequest struct {
	// Display name.
	//
	// Must be unique among the account's product lines; a duplicate name returns a conflict error.
	Name string `json:"name" validate:"required,max=255"`
	// ID of the unit group to associate with this product line.
	//
	// The unit group determines the set of units available to products in this product line.
	UnitGroupID string `json:"unit_group_id" validate:"required"`
	// Default commission policy for products in this product line.
	//
	// - `commission_exempt`: no commission applies to these products.
	// - `commission_applied`: commission applies to these products, unless overridden elsewhere.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Default freight policy for products in this product line.
	//
	// - `free_freight`: these products do not incur a freight charge.
	// - `billed_freight`: freight is billed for these products, unless overridden elsewhere.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
}

var sampleCreateProductLineRequest = &CreateProductLineRequest{
	Name:             apiresource.SampleProductLineName,
	UnitGroupID:      apiresource.SampleUnitGroupID,
	CommissionPolicy: constants.CommissionPolicyExempt,
	FreightPolicy:    constants.FreightPolicyBilled,
}

func (*CreateProductLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductLineRequest)
}

// Creates an account-owned product line.
type CreateProductLineEndpoint struct{}

func (e *CreateProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductLineRequest, *apiresource.ProductLine] {
	return (&apiendpoint.APIEndpoint[*CreateProductLineRequest, *apiresource.ProductLine]{
		Title:               "Create Product Line",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/catalog/product-lines",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).CreateProductLine
		},
		LocationFunc: func(resp *apiresource.ProductLine) string {
			return "/v1/catalog/product-lines/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group"},
		}),
	})
}
