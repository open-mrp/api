package machineep

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

// Request to suggest items from a machine's scan history.
type SuggestMachineItemsFromScansRequest struct {
	// ID of the machine.
	MachineID string `path:"id" validate:"required"`
	// How many months of scan history to consider.
	//
	// Defaults to 24 when omitted.
	WindowMonths field.Optional[int32] `json:"window_months,omitzero" validate:"omitempty,min=1,max=120"`
}

var sampleSuggestMachineItemsFromScansRequest = &SuggestMachineItemsFromScansRequest{
	WindowMonths: field.Some(int32(24)),
}

func (*SuggestMachineItemsFromScansRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSuggestMachineItemsFromScansRequest)
}

// Suggests items from a machine's scan history without changing its configured set.
//
// Use the response to curate a list, then save with Replace Machine Items. Suggestions are ordered by most recent scan.
type SuggestMachineItemsFromScansEndpoint struct{}

func (e *SuggestMachineItemsFromScansEndpoint) Materialize() *apiendpoint.APIEndpoint[*SuggestMachineItemsFromScansRequest, *apiresource.List[apiresource.MachineItemScanSuggestion]] {
	return (&apiendpoint.APIEndpoint[*SuggestMachineItemsFromScansRequest, *apiresource.List[apiresource.MachineItemScanSuggestion]]{
		Title:             "Suggest Machine Items From Scans",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}/items/actions/suggest-from-scans",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ReadOnly:          true,
		ObjectType:        constants.ObjectTypeMachineItemScanSuggestion,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SuggestMachineItemsFromScansRequest) (*apiresource.List[apiresource.MachineItemScanSuggestion], *apierror.APIError) {
			return svc.(MachineSvc).SuggestMachineItemsFromScans
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeMachineItemScanSuggestion,
			Fields:     []string{"item"},
		}),
	})
}
