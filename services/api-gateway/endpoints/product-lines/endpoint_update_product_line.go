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

// Request to update a product line.
type UpdateProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
	// Display name, such as `Outdoor`.
	//
	// Unique (case-insensitive) among your account's lines and the system lines.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Description of the line, shown to customers in the portal. Send `null` to remove it.
	Description field.Clearable[string] `json:"description,omitzero" validate:"omitempty,max=5000"`
	// Whether the line can be newly assigned: `active` or `archived`.
	//
	// Archiving keeps the line on its products and customers; it just can't be assigned or granted anew.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// Whether sales of this line's products earn commission: `applied` or `exempt`.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Whether this line's products add a freight charge: `billed` or `free`.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// Where this line's items come from when they don't say for themselves: `stock`, `make` or `buy`.
	//
	// Send `null` to use the account default.
	Supply field.Clearable[constants.Supply] `json:"supply,omitzero"`
	// Unit group the line's products are priced in.
	UnitGroupID field.Optional[string] `json:"unit_group_id,omitzero" visibility:"internal"`
}

var sampleUpdateProductLineRequest = &UpdateProductLineRequest{
	Name:             field.Some("Outdoor Fasteners"),
	CommissionPolicy: field.Some(constants.CommissionPolicyApplied),
	Supply:           field.Set(constants.SupplyMake),
}

func (*UpdateProductLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductLineRequest)
}

// Updates a product line.
//
// Only fields you send are changed.
type UpdateProductLineEndpoint struct{}

func (e *UpdateProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductLineRequest, *apiresource.ProductLine] {
	return (&apiendpoint.APIEndpoint[*UpdateProductLineRequest, *apiresource.ProductLine]{
		Title:             "Update Product Line",
		Method:            http.MethodPatch,
		Route:             productLineRoute,
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).UpdateProductLine
		},
		IncludeConfig: productLineIncludes(),
	})
}
