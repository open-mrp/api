package partep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to export parts as an Excel file.
type ExportPartsRequest struct {
	// Free-text search query matched against parts.
	Query *string `query:"q"`
	// Filter by category IDs.
	CategoryIDs []string `query:"category_ids"`
	// Filter by attribute IDs.
	AttributeIDs []string `query:"attribute_ids"`
	// Start of creation date range.
	StartDate *time.Time `query:"start_date"`
	// End of creation date range.
	EndDate *time.Time `query:"end_date"`
}

// Exports all matching parts as an Excel file.
type ExportPartsEndpoint struct{}

func (e *ExportPartsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportPartsRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportPartsRequest, *httptransport.FileDownload]{
		Title:               "Export Parts",
		Method:              http.MethodGet,
		ContentType:         "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:               "/v1/catalog/parts/actions/export",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainParts, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportPartsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(PartSvc).ExportParts
		},
	})
}
