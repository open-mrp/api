package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// AnalyzeCustomerPricingRequest is the request to audit contracted customer prices.
type AnalyzeCustomerPricingRequest struct {
	// Restrict the analysis to these customers. Omit to cover every customer with a contracted price.
	//
	// Peer medians are still computed across all customers, so narrowing the result does not change what a price is compared against.
	CustomerIDs []string `json:"customer_ids,omitzero"`
	// Restrict the analysis to customers in these customer groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// The gross margin a price is expected to clear, as a fraction between 0 and 1.
	TargetGrossMargin field.Optional[string] `json:"target_gross_margin,omitzero" format:"decimal"`
	// How far below the peer median a price must sit to be flagged, as a fraction between 0 and 1.
	OutlierTolerance field.Optional[string] `json:"outlier_tolerance,omitzero" format:"decimal"`
}

// Flags contracted customer prices that are unusually low or unprofitable.
//
// Two independent checks run over every account price. The first compares a price against the median price other customers pay for the same product line and attributes — the same pair the pricing engine matches on — so a price is only ever compared against prices that buy the same thing. The second computes gross margin from the cost of the products the price applies to. A price may be flagged by either or both.
//
// Prices a customer receives through its parent account are included and marked, since they are easy to miss when auditing customer by customer. Manual price overrides entered on an individual order are not visible here: they bypass contracted pricing entirely and are only recorded on the order line.
type AnalyzeCustomerPricingEndpoint struct{}

func (e *AnalyzeCustomerPricingEndpoint) Materialize() *apiendpoint.APIEndpoint[*AnalyzeCustomerPricingRequest, *apiresource.AnalyzeCustomerPricingResponse] {
	return (&apiendpoint.APIEndpoint[*AnalyzeCustomerPricingRequest, *apiresource.AnalyzeCustomerPricingResponse]{
		Title:               "Analyze Customer Pricing",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/customer-pricing",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead}},
		// The roots the include resolver walks are the findings, not the response wrapper, so ObjectType names the finding and ExtractRoots reaches into the list.
		ObjectType:    constants.ObjectTypeCustomerPricingFinding,
		IncludeConfig: customerPricingIncludeConfig(),
		ServiceHandler: func(svc any) func(ctx context.Context, req *AnalyzeCustomerPricingRequest) (*apiresource.AnalyzeCustomerPricingResponse, *apierror.APIError) {
			return svc.(AnalyticsSvc).AnalyzeCustomerPricing
		},
	})
}

func customerPricingIncludeConfig() *apiendpoint.IncludeConfig {
	cfg := apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeCustomerPricingFinding,
		Fields:     []string{"customer", "product_line", "attributes", "unit_price.numerator_unit", "unit_price.denominator_unit"},
	})
	// Roots must be addressable: the resolver writes the loaded relation back onto each finding.
	cfg.ExtractRoots = func(resp any) []any {
		out := resp.(*apiresource.AnalyzeCustomerPricingResponse)
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
