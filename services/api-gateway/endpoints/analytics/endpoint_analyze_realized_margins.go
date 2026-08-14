package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// AnalyzeRealizedMarginsRequest is the request to audit what customers were actually charged.
type AnalyzeRealizedMarginsRequest struct {
	// Start of the invoiced window.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// End of the invoiced window.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// Restrict the result to these customers.
	//
	// Peer medians are still computed across every customer that bought the SKU, so narrowing the result does not change what a price is compared against.
	CustomerIDs []string `json:"customer_ids,omitzero"`
	// Restrict the result to customers in these customer groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Restrict the result to these product lines.
	ProductLineIDs []string `json:"product_line_ids,omitzero"`
	// The gross margin a sale is expected to clear, as a fraction between 0 and 1.
	TargetGrossMargin field.Optional[string] `json:"target_gross_margin,omitzero" format:"decimal"`
	// How far below the peer median an achieved price must sit to be flagged, as a fraction between 0 and 1.
	OutlierTolerance field.Optional[string] `json:"outlier_tolerance,omitzero" format:"decimal"`
}

// Flags what customers were actually charged, as opposed to what they are contracted to be charged.
//
// Invoiced lines over the window are rolled up to one row per customer and SKU, weighted by quantity, and each row is checked twice: against the median price other customers achieved on the same SKU, and against a target gross margin computed from the cost captured on the lines. Findings are ranked by money at stake rather than by percentage, so a thin margin on a large account outranks a worse percentage on a single small order.
//
// This is the only view that sees a price typed onto an individual order. A manual line override bypasses contracted prices and volume discounts entirely, so it never appears in an audit of configured pricing.
type AnalyzeRealizedMarginsEndpoint struct{}

func (e *AnalyzeRealizedMarginsEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeRealizedMarginsRequest, *apiresource.AnalyzeRealizedMarginsResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeRealizedMarginsRequest, *apiresource.AnalyzeRealizedMarginsResponse]{
		Title:             "Analyze Realized Margins",
		Method:            http.MethodPut,
		Route:             "/v1/core/analytics/realized-margins",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		AgentTool:         true,
		Preview:           true,
		// Invoiced revenue per customer is exactly what Analyze Sales gates behind invoices:read, and this endpoint reads the same rows. Permissions are any-of, so listing discounts:read alongside it would let a discount reader see invoiced revenue they cannot read directly.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		// The roots the include resolver walks are the findings, not the response wrapper.
		ObjectType:    constants.ObjectTypeRealizedMarginFinding,
		IncludeConfig: realizedMarginIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeRealizedMarginsRequest) (*apiresource.AnalyzeRealizedMarginsResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeRealizedMargins
		},
	})
}

func realizedMarginIncludeConfig() *apiendpoint.IncludeConfig {
	cfg := apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeRealizedMarginFinding,
		Fields:     []string{"customer", "customer_group", "item", "product_line"},
	})
	cfg.ExtractRoots = func(resp any) []any {
		out := resp.(*apiresource.AnalyzeRealizedMarginsResponse)
		if out.Findings == nil {
			return nil
		}
		roots := make([]any, 0, len(out.Findings.Data))
		for i := range out.Findings.Data {
			roots = append(roots, &out.Findings.Data[i])
		}
		return roots
	}
	return cfg
}
