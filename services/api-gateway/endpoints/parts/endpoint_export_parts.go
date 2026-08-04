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
	// Free-text search term matched against the part's SKU or description.
	Query *string `query:"q"`
	// Only return parts belonging to any of these item categories.
	CategoryIDs []string `query:"category_ids"`
	// Only return parts carrying at least one of these attributes.
	AttributeIDs []string `query:"attribute_ids"`
	// Only return parts created at or after this time.
	StartDate *time.Time `query:"start_date"`
	// Only return parts created at or before this time.
	EndDate *time.Time `query:"end_date"`
}

// Exports the parts matching the given filters as an Excel workbook.
//
// The workbook holds one row per part with its ID, SKU, description, category, and unit price and unit cost alongside the units they are quoted in, followed by one column for each property defined on the exported parts' categories. Every match is exported in a single file, so this endpoint is not paginated.
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
