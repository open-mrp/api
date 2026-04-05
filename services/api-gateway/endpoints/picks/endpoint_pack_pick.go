package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// PackPickRequest is the request to pack a pick and create a shipment.
type PackPickRequest struct {
	// The ID of the pick to pack.
	PickID string `path:"id" validate:"required"`
	// The number of cases for the shipment.
	ShipmentCaseCount int32 `json:"shipment_case_count" validate:"required,gte=1"`
}

var samplePackPickRequest = &PackPickRequest{
	ShipmentCaseCount: 3,
}

func (*PackPickRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePackPickRequest)
}

type PackPickEndpoint struct{}

func (e *PackPickEndpoint) Materialize() *apiendpoint.APIEndpoint[*PackPickRequest, *apiresource.PackPickResponse] {
	return &apiendpoint.APIEndpoint[*PackPickRequest, *apiresource.PackPickResponse]{
		Title:             "Pack Pick",
		Description:       "Packs a pick and creates a shipment from the picked lines.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/picks/{id}/actions/pack",
		ContentType:       "application/json",
		Request:           &PackPickRequest{},
		Response:          &apiresource.PackPickResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *PackPickRequest) (*apiresource.PackPickResponse, *apierror.APIError) {
			return svc.(PickSvc).PackPick
		},
	}
}
