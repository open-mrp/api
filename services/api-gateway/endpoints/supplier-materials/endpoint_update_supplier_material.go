package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a supplier material.
type UpdateSupplierMaterialRequest struct {
	// Supplier ID.
	SupplierID string `path:"supplier_id" validate:"required"`
	// Supplier material ID.
	MaterialID string `path:"id" validate:"required"`
	// Supplier part number for this material.
	SupplierPartNumber field.Optional[string] `json:"supplier_part_number,omitzero" validate:"omitempty,max=255"`
	// Supplier description for this material.
	SupplierDescription field.Optional[string] `json:"supplier_description,omitzero" validate:"omitempty,max=255"`
	// Active status.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).UpdateSupplierMaterial
		},
	})
}
