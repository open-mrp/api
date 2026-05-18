package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdatePickRequest is the request to partially update a pick's metadata.
type UpdatePickRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
	// Pick number.
	Number *string `json:"number,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Timestamp when the pick was finished. Pass an empty string to clear.
	FinishedAt *string `json:"finished_at,omitempty" nullable:"true"`
}

var sampleUpdatePickNumber = "PCK-2025-0042"
var sampleUpdatePickRequest = &UpdatePickRequest{
	Number: &sampleUpdatePickNumber,
}

func (*UpdatePickRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePickRequest)
}

// Partially updates a pick's metadata.
type UpdatePickEndpoint struct{}

func (e *UpdatePickEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePickRequest, *apiresource.PickDetail] {
	return (&apiendpoint.APIEndpoint[*UpdatePickRequest, *apiresource.PickDetail]{
		Title:             "Update Pick",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePickRequest) (*apiresource.PickDetail, *apierror.APIError) {
			return svc.(PickSvc).UpdatePick
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
			Fields:     []string{"lines"},
		}),
	})
}
