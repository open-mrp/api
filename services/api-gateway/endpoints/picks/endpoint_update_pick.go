package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update a pick's metadata.
type UpdatePickRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
	// New number to assign to the pick.
	//
	// Maximum 255 characters. Renaming a pick does not rename the sales order it was created from.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Timestamp when the pick was finished, in RFC 3339 format.
	//
	// Setting it closes the pick out even if lines are still unpacked; send `null` to clear it and reopen the pick.
	FinishedAt field.Clearable[string] `json:"finished_at,omitzero"`
}

var sampleUpdatePickNumber = "PCK-2025-0042"
var sampleUpdatePickRequest = &UpdatePickRequest{
	Number: field.Some(sampleUpdatePickNumber),
}

func (*UpdatePickRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdatePickRequest)
}

// Partially updates a pick's metadata.
//
// Only the fields provided in the request are changed. This endpoint edits the pick record itself; use the pick and pack actions to change what has actually been picked.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdatePickRequest) (*apiresource.Pick, *apierror.APIError) {
			return svc.(PickSvc).UpdatePick
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
			Fields:     []string{"lines"},
		}),
	})
}
