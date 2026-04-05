package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type UpdateMaterialRequest struct {
	ItemID      string                `path:"id" validate:"required"`
	SKU         *string               `json:"sku,omitempty"`
	Description *string               `json:"description,omitempty"`
	Notes       *string               `json:"notes,omitempty"`
	OrderPoint  *QuantityInputRequest `json:"order_point,omitempty"`
	LeadTime    *QuantityInputRequest `json:"lead_time,omitempty"`
}

var sampleUpdateMaterialSKU = "MAT-001-UPDATED"
var sampleUpdateMaterialRequest = &UpdateMaterialRequest{
	SKU: &sampleUpdateMaterialSKU,
}

func (*UpdateMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMaterialRequest)
}

type UpdateMaterialEndpoint struct{}

func (e *UpdateMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMaterialRequest, *apiresource.Material] {
	return &apiendpoint.APIEndpoint[*UpdateMaterialRequest, *apiresource.Material]{
		Title:             "Update Material",
		Description:       "Partially updates a material.",
		Method:            http.MethodPatch,
		Route:             "/v1/operations/materials/{id}",
		Request:           &UpdateMaterialRequest{},
		Response:          &apiresource.Material{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).UpdateMaterial
		},
	}
}
