package resourceloaders

import (
	"context"
	"strings"

	"github.com/open-mrp/apikit/include"
	"google.golang.org/grpc"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

var priceListLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.price_list")

// priceListClient is the CorePriceListService client. Set at startup.
var priceListClient pb.CorePriceListServiceClient

// SetPriceListClient is called once at startup with the CorePriceListService client.
func SetPriceListClient(c pb.CorePriceListServiceClient) {
	priceListClient = c
}

// Load meta keys a price list or rule carries for its includes.
const (
	PriceListMetaRuleIDs         = "rule_ids"
	PriceListMetaRulesNextCursor = "rules_next_cursor"
	PriceListRuleMetaItemIDs     = "item_ids"
	PriceListRuleMetaLineIDs     = "product_line_ids"
	PriceListRuleMetaCategoryIDs = "category_ids"
	PriceListRuleMetaAttrIDs     = "attribute_ids"
	PriceListRuleMetaListID      = "price_list_id"
)

// PriceListRulesPath is the list of one price list's rules, where its embedded rules continue.
func PriceListRulesPath(listID string) string {
	return "/v1/sales/price-lists/" + listID + "/rules"
}

// rulesRequested reports whether any requested include expands a price list's rules, so lists load their first rule
// IDs only when asked.
func rulesRequested(ctx context.Context) bool {
	for key := range include.RequestedIncludeSet(ctx) {
		if key == "rules" || strings.HasPrefix(key, "rules.") || strings.HasSuffix(key, ".rules") || strings.Contains(key, ".rules.") {
			return true
		}
	}
	return false
}

// LoadPriceLists batch-loads price lists, with their first rule IDs when an include expands rules.
func LoadPriceLists(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, priceListLoaderTracer, "loader.price_lists.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPriceListsByIDsResponse, error) {
			return priceListClient.BatchGetPriceListsByIDs(ctx, &pb.BatchGetPriceListsByIDsRequest{Ids: ids, WithRules: rulesRequested(ctx)}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.PriceLists))
	for _, l := range resp.PriceLists {
		out[l.Id] = PriceListFromProto(l)
		meta.Set(constants.ObjectTypePriceList, l.Id, PriceListMetaRuleIDs, l.RuleIds)
		meta.Set(constants.ObjectTypePriceList, l.Id, PriceListMetaRulesNextCursor, l.RulesNextCursor)
	}
	return out, nil
}

// PriceListFromProto is a price list as the response shows it, rules unexpanded.
func PriceListFromProto(l *pb.PriceListInfo) *apiresource.PriceList {
	return &apiresource.PriceList{
		ID:          l.Id,
		Object:      constants.ObjectTypePriceList,
		Status:      constants.ArchiveStatus(l.Status),
		Name:        l.Name,
		Description: l.Description,
		Currency:    constants.Currency(l.Currency),
		CreatedAt:   grpcutil.TimestampToTime(l.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(l.UpdatedAt),
	}
}

// LoadPriceListRules batch-loads price-list rules.
func LoadPriceListRules(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, priceListLoaderTracer, "loader.price_list_rules.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPriceListRulesByIDsResponse, error) {
			return priceListClient.BatchGetPriceListRulesByIDs(ctx, &pb.BatchGetPriceListRulesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Rules))
	for _, r := range resp.Rules {
		out[r.Id] = PriceListRuleFromProto(ctx, r)
	}
	return out, nil
}

// PriceListRuleFromProto is a rule as the response shows it, with its scope IDs stashed for the scope includes.
func PriceListRuleFromProto(ctx context.Context, r *pb.PriceListRuleInfo) *apiresource.PriceListRule {
	meta := include.GetLoadMeta(ctx)
	scope := r.GetScope()
	meta.Set(constants.ObjectTypePriceListRule, r.Id, PriceListRuleMetaListID, r.PriceListId)
	meta.Set(constants.ObjectTypePriceListRule, r.Id, PriceListRuleMetaItemIDs, scope.GetItemIds())
	meta.Set(constants.ObjectTypePriceListRule, r.Id, PriceListRuleMetaLineIDs, scope.GetProductLineIds())
	meta.Set(constants.ObjectTypePriceListRule, r.Id, PriceListRuleMetaCategoryIDs, scope.GetCategoryIds())
	meta.Set(constants.ObjectTypePriceListRule, r.Id, PriceListRuleMetaAttrIDs, scope.GetAttributeIds())
	return &apiresource.PriceListRule{
		ID:        r.Id,
		Object:    constants.ObjectTypePriceListRule,
		Scope:     &apiresource.PriceListRuleScope{},
		Price:     priceListPriceFromProto(r.GetPrice()),
		CreatedAt: grpcutil.TimestampToTime(r.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(r.UpdatedAt),
	}
}

func priceListPriceFromProto(p *pb.PriceListPrice) *apiresource.PriceListPrice {
	out := &apiresource.PriceListPrice{
		Type:          constants.PriceListPriceType(p.GetType()),
		UnitPrice:     ExactUnitPriceFromProto(p.GetUnitPrice()),
		PercentOfBase: p.PercentOfBase,
		Unit:          apiresource.UnitStub(p.GetUnitId()),
		Tiers:         []apiresource.PriceListTier{},
	}
	for _, t := range p.GetTiers() {
		out.Tiers = append(out.Tiers, apiresource.PriceListTier{
			MinQuantity:   t.MinQuantity,
			PercentOfBase: t.PercentOfBase,
			UnitPrice:     ExactUnitPriceFromProto(t.UnitPrice),
		})
	}
	return out
}
