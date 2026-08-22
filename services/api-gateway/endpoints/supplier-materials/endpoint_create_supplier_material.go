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

// Request to create a supplier material.
type CreateSupplierMaterialRequest struct {
	// ID of the supplier to link the material to.
	SupplierID string `path:"supplier_id" validate:"required"`
	// ID of the material the supplier provides.
	//
	// A material can be linked to a given supplier at most once; creating a duplicate link fails with a conflict error.
	MaterialID string `json:"material_id" validate:"required"`
	// The part number the supplier uses for this material in their own catalog.
	SupplierPartNumber string `json:"supplier_part_number" validate:"required,max=255"`
	// The supplier's own description of this material.
	SupplierDescription field.Optional[string] `json:"supplier_description,omitzero" validate:"omitempty,max=255"`
	// Whether this supplier is currently one you would source the material from.
	//
	// Links are created active unless this is explicitly set to `false`.
	IsActive field.Optional[bool] `json:"is_active,omitzero"`
}

var sampleIsActive = true

var sampleCreateSupplierMaterialRequest = &CreateSupplierMaterialRequest{
	MaterialID:         apiresource.SampleMaterialID,
	SupplierPartNumber: "SUP-PART-001",
	IsActive:           field.Some(sampleIsActive),
}

func (*CreateSupplierMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSupplierMaterialRequest)
}

// Links a material to a supplier, recording the supplier's part number and description for it.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).CreateSupplierMaterial
		},
	})
}
