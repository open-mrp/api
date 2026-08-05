package materialep

import (
	"context"
	"net/http"
	"time"

	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to export materials as an Excel file.
type ExportMaterialsRequest struct {
	// Free-text search term matched against material SKU and description.
	Query *string `query:"q"`
	// Filter to materials in any of these categories.
	CategoryIDs []string `query:"category_ids"`
	// Filter to materials carrying any of these attributes.
	AttributeIDs []string `query:"attribute_ids"`
	// Filter to materials created on or after this date.
	StartDate *time.Time `query:"starts_at"`
	// Filter to materials created on or before this date.
	EndDate *time.Time `query:"ends_at"`
}

// Downloads the materials matching the given filters as an Excel workbook named `materials.xlsx`.
//
// The filters and ordering work the same way as on the material list endpoint, but the export is not paginated: every match lands in a single `Materials` sheet, one row per material. Columns cover the ID, SKU, description, category, and the unit price and unit cost with their units, plus one column for each property defined on the exported materials' categories, filled in from their attributes.
type ExportMaterialsEndpoint struct{}

func (e *ExportMaterialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ExportMaterialsRequest, *httptransport.FileDownload] {
	return (&apiendpoint.APIEndpoint[*ExportMaterialsRequest, *httptransport.FileDownload]{
		Title:               "Export Materials",
		Method:              http.MethodGet,
		ContentType:         "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Route:               "/v1/catalog/materials/actions/export",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMaterials, Action: types.ActionRead}, {Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ExportMaterialsRequest) (*httptransport.FileDownload, *apierror.APIError) {
			return svc.(MaterialSvc).ExportMaterials
		},
	})
}
