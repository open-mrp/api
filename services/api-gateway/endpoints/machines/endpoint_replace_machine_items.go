package machineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// Request to replace the items a machine is configured to run.
type ReplaceMachineItemsRequest struct {
	// ID of the machine.
	MachineID string `path:"id" validate:"required"`
	// The full set of item IDs this machine should run.
	//
	// Replaces the existing set; send an empty list to clear all configured items.
	ItemIDs []string `json:"item_ids,omitzero"`
}

var sampleReplaceMachineItemsRequest = &ReplaceMachineItemsRequest{
	ItemIDs: []string{apiresource.SampleItemID},
}

func (*ReplaceMachineItemsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleReplaceMachineItemsRequest)
}

// Replaces the items a machine is configured to run.
//
// This is a full replacement, not a merge: items omitted from the request are removed. Production schedule generation uses this set for SKU discovery and machine eligibility.
type ReplaceMachineItemsEndpoint struct{}

func (e *ReplaceMachineItemsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ReplaceMachineItemsRequest, *apiresource.List[apiresource.Item]] {
	return (&apiendpoint.APIEndpoint[*ReplaceMachineItemsRequest, *apiresource.List[apiresource.Item]]{
		Title:             "Replace Machine Items",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/machines/{id}/items",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeItem,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainMachines, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ReplaceMachineItemsRequest) (*apiresource.List[apiresource.Item], *apierror.APIError) {
			return svc.(MachineSvc).ReplaceMachineItems
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeItem,
			Fields:     []string{"category", "unit_value", "attributes"},
		}),
	})
}
