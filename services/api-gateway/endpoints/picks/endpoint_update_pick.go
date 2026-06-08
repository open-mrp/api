package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// UpdatePickRequest is the request to partially update a pick's metadata.
type UpdatePickRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
	// Pick number.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Timestamp when the pick was finished. Pass an empty string to clear.
	FinishedAt field.Optional[string] `json:"finished_at,omitzero"`
}

var sampleUpdatePickNumber = "PCK-2025-0042"
var sampleUpdatePickRequest = &UpdatePickRequest{
	Number: field.Some(sampleUpdatePickNumber),
}

func (*UpdatePickRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePickRequest)
}

// Partially updates a pick's metadata.
type UpdatePickEndpoint struct{}

func (e *UpdatePickEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdatePickRequest, *apiresource.Pick] {
	return (&apiendpoint.APIEndpoint[*UpdatePickRequest, *apiresource.Pick]{
		Title:             "Update Pick",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePick,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePickRequest) (*apiresource.Pick, *apierror.APIError) {
			return svc.(PickSvc).UpdatePick
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
			Fields:     []string{"lines"},
		}),
	})
}
