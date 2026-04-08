package suppliermaterialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type UpdateSupplierMaterialRequest struct {
	SupplierID          string  `path:"supplier_id" validate:"required"`
	ItemID              string  `path:"id" validate:"required"`
	SupplierPartNumber  *string `json:"supplier_part_number,omitempty" validate:"omitempty,max=255"`
	SupplierDescription *string `json:"supplier_description,omitempty" validate:"omitempty,max=255"`
	IsActive            *bool   `json:"is_active,omitempty"`
}

var sampleUpdateSupplierPartNumber = "SUP-PART-002"
var sampleUpdateSupplierMaterialRequest = &UpdateSupplierMaterialRequest{
	SupplierPartNumber: &sampleUpdateSupplierPartNumber,
}

func (*UpdateSupplierMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateSupplierMaterialRequest)
}

type UpdateSupplierMaterialEndpoint struct{}

func (e *UpdateSupplierMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateSupplierMaterialRequest, *apiresource.SupplierMaterial] {
	return &apiendpoint.APIEndpoint[*UpdateSupplierMaterialRequest, *apiresource.SupplierMaterial]{
		Title:             "Update Supplier Material",
		Description:       "Partially updates a supplier material.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/suppliers/{supplier_id}/materials/{id}",
		Request:           &UpdateSupplierMaterialRequest{},
		Response:          &apiresource.SupplierMaterial{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateSupplierMaterialRequest) (*apiresource.SupplierMaterial, *apierror.APIError) {
			return svc.(SupplierMaterialSvc).UpdateSupplierMaterial
		},
	}
}
