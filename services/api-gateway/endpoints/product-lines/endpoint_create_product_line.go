package productlineep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to create a product line.
type CreateProductLineRequest struct {
	// Display name, such as `Outdoor`.
	//
	// Unique (case-insensitive) among your account's lines and the system lines.
	Name string `json:"name" validate:"required,max=255"`
	// Description of the line, shown to customers in the portal.
	Description field.Optional[string] `json:"description,omitzero" validate:"omitempty,max=5000"`
	// Whether the line can be newly assigned: `active` or `archived`. Defaults to `active`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Whether sales of this line's products earn commission: `applied` or `exempt`.
	//
	// A customer's or account group's policy can override it.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// Whether this line's products add a freight charge: `billed` or `free`.
	//
	// A customer's or account group's policy can override it.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
	// Where this line's items come from when they don't say for themselves: `stock`, `make` or `buy`.
	//
	// Omit it to use the account default.
	Supply field.Optional[constants.Supply] `json:"supply,omitzero"`
	// Unit group the line's products are priced in. Omit it to price them in their category's group.
	UnitGroupID field.Optional[string] `json:"unit_group_id,omitzero" visibility:"internal"`
}

var sampleCreateProductLineRequest = &CreateProductLineRequest{
	Name:             apiresource.SampleProductLineName,
	Description:      field.Some("Bolts, screws, and anchors for heavy industrial assembly."),
	CommissionPolicy: constants.CommissionPolicyExempt,
	FreightPolicy:    constants.FreightPolicyBilled,
	Supply:           field.Some(constants.SupplyStock),
}

func (*CreateProductLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductLineRequest)
}

// Creates a product line.
//
// The new line belongs to your account and starts with no products. Set a product's line to add it. Customers and account groups can only be granted lines your account owns.
type CreateProductLineEndpoint struct{}

func (e *CreateProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductLineRequest, *apiresource.ProductLine] {
	return (&apiendpoint.APIEndpoint[*CreateProductLineRequest, *apiresource.ProductLine]{
		Title:             "Create Product Line",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             productLinesRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).CreateProductLine
		},
		LocationFunc: func(resp *apiresource.ProductLine) string {
			return productLinesRoute + "/" + resp.ID
		},
		IncludeConfig: productLineIncludes(),
	})
}
