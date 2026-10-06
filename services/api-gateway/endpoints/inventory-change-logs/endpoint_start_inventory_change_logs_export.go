package inventorychangelogep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Filters which inventory change logs land in the exported file.
type StartInventoryChangeLogsExportRequest struct {
	// Restricts the file to changes affecting these items.
	ItemIDs []string `json:"item_ids,omitzero"`
	// Restricts the file to these action types.
	ActionTypes []constants.InventoryActionType `json:"action_types,omitzero"`
	// Restricts the file to changes made by these users.
	//
	// Changes that were recorded without a responsible user are excluded whenever this filter is set.
	ChangedByUserIDs []string `json:"changed_by_user_ids,omitzero"`
	// Restricts the file to change logs created on or after this timestamp.
	//
	// Unlike the list, no default window applies: leave it out to export from the account's first change.
	StartsAt field.Optional[time.Time] `json:"starts_at,omitzero"`
	// Restricts the file to change logs created on or before this timestamp.
	EndsAt field.Optional[time.Time] `json:"ends_at,omitzero"`
}

var sampleStartInventoryChangeLogsExportRequest = &StartInventoryChangeLogsExportRequest{
	ActionTypes: []constants.InventoryActionType{constants.InventoryActionTypeScan},
	StartsAt:    field.Some(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
	EndsAt:      field.Some(time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)),
}

func (*StartInventoryChangeLogsExportRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleStartInventoryChangeLogsExportRequest)
}

// Starts an export of every inventory change log the filters select and returns the job that tracks it.
//
// Poll the job; once it completes, `export.url` links to the Excel file. The file has one row per change log, newest first, with the same columns as the synchronous export, and is named for the window you asked for — `inventory-change-logs-<starts_at>-<ends_at>.xlsx`, each bound as its UTC date and `all` in place of a bound you left open. A file with more change logs than one worksheet holds continues on further worksheets.
type StartInventoryChangeLogsExportEndpoint struct{}

func (e *StartInventoryChangeLogsExportEndpoint) Materialize() *apiendpoint.APIEndpoint[*StartInventoryChangeLogsExportRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*StartInventoryChangeLogsExportRequest, *apiresource.Job]{
		Title:             "Start Inventory Change Logs Export",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/inventory-change-logs/actions/export",
		SDKMethodKey:      "start_export",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeJob,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainInventoryLogs, Action: types.ActionRead},
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *StartInventoryChangeLogsExportRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(InventoryChangeLogSvc).StartInventoryChangeLogsExport
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
