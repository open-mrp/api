package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a supplier material.
type UpdateSupplierMaterialRequest struct {
	// ID of the supplier the material is linked to.
	SupplierID string `path:"supplier_id" validate:"required"`
	// ID of the material whose supplier link to update.
	//
	// Supplier materials are addressed by the combination of supplier and material, so this path parameter takes the material's ID.
	MaterialID string `path:"id" validate:"required"`
	// New part number the supplier uses for this material.
	SupplierPartNumber field.Optional[string] `json:"supplier_part_number,omitzero" validate:"omitempty,max=255"`
	// New supplier description of this material.
	SupplierDescription field.Optional[string] `json:"supplier_description,omitzero" validate:"omitempty,max=255"`
	// Whether this supplier is currently one you would source the material from.
	IsActive field.Optional[bool] `json:"is_active,omitzero"`
}

var sampleUpdateSupplierPartNumber = "SUP-PART-002"
var sampleUpdateSupplierMaterialRequest = &UpdateSupplierMaterialRequest{
	SupplierPartNumber: field.Some(sampleUpdateSupplierPartNumber),
}

func (*UpdateSupplierMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSupplierMaterialRequest)
}

// Partially updates a supplier material.
//
// Fields not provided retain their current values.
type UpdateSupplierMaterialEndpoint struct{}

func (e *UpdateSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return (&apiendpoint.APIEndpoint[*UpdateSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Update Supplier Material",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).UpdateSupplierMaterial
		},
	})
}
