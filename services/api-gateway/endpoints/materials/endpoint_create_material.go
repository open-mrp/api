package materialep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// QuantityInputRequest is a quantity value and unit.
type QuantityInputRequest struct {
	// Quantity value.
	Value string `json:"value" validate:"required"`
	// Unit ID.
	UnitID string `json:"unit_id" validate:"required,max=191"`
}

// Request to create a material.
type CreateMaterialRequest struct {
	// SKU code.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description.
	Description *string `json:"description,omitempty"`
	// Notes.
	Notes *string `json:"notes,omitempty"`
	// Category ID.
	CategoryID string `json:"category_id" validate:"required,max=191"`
	// Order point quantity.
	OrderPoint *QuantityInputRequest `json:"order_point,omitempty"`
	// Lead time quantity.
	LeadTime *QuantityInputRequest `json:"lead_time,omitempty"`
}

var sampleCreateMaterialRequest = &CreateMaterialRequest{
	SKU:        "MAT-001",
	CategoryID: apiresource.SampleItemCategoryID,
}

func (*CreateMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMaterialRequest)
}

type CreateMaterialEndpoint struct{}

func (e *CreateMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMaterialRequest, *apiresource.Material] {
	return &apiendpoint.APIEndpoint[*CreateMaterialRequest, *apiresource.Material]{
		Title:             "Create Material",
		Description:       "Creates a material.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/materials",
		Request:           &CreateMaterialRequest{},
		Response:          &apiresource.Material{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMaterialRequest) (*apiresource.Material, *apierror.APIError) {
			return svc.(MaterialSvc).CreateMaterial
		},
		LocationFunc: func(resp *apiresource.Material) string {
			return "/v1/operations/materials/" + resp.ID
		},
	}
}
