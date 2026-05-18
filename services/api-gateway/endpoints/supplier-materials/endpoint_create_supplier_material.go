package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a supplier material.
type CreateSupplierMaterialRequest struct {
	// Supplier ID.
	SupplierID string `path:"supplier_id" validate:"required"`
	// Material ID.
	MaterialID string `json:"material_id" validate:"required"`
	// Supplier part number for this material.
	SupplierPartNumber string `json:"supplier_part_number" validate:"required,max=255"`
	// Supplier description for this material.
	SupplierDescription *string `json:"supplier_description,omitempty" validate:"omitempty,max=255"`
	// Active status.
	IsActive *bool `json:"is_active"`
}

var sampleIsActive = true

var sampleCreateSupplierMaterialRequest = &CreateSupplierMaterialRequest{
	MaterialID:         apiresource.SampleMaterialID,
	SupplierPartNumber: "SUP-PART-001",
	IsActive:           &sampleIsActive,
}

func (*CreateSupplierMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSupplierMaterialRequest)
}

// Creates a supplier material association.
type CreateSupplierMaterialEndpoint struct{}

func (e *CreateSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return (&apiendpoint.APIEndpoint[*CreateSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Create Supplier Material",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/suppliers/{supplier_id}/materials",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).CreateSupplierMaterial
		},
	})
}
