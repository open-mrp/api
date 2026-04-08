package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type CreateSupplierMaterialRequest struct {
	SupplierID          string  `path:"supplier_id" validate:"required"`
	MaterialID          string  `json:"material_id" validate:"required,max=191"`
	SupplierPartNumber  string  `json:"supplier_part_number" validate:"required,max=255"`
	SupplierDescription *string `json:"supplier_description,omitempty" validate:"omitempty,max=255"`
	IsActive            *bool   `json:"is_active"`
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

type CreateSupplierMaterialEndpoint struct{}

func (e *CreateSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return &apiendpoint.APIEndpoint[*CreateSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Create Supplier Material",
		Description:       "Creates a new supplier material association.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/suppliers/{supplier_id}/materials",
		Request:           &CreateSupplierMaterialRequest{},
		Response:          &apiresource.SupplierMaterial{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).CreateSupplierMaterial
		},
	}
}
