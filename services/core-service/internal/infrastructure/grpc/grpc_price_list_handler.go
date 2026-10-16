package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
)

type priceListGRPCHandler struct {
	pb.UnimplementedCorePriceListServiceServer

	svc domain.PriceListSvc
}

func RegisterPriceListService(server *grpc.Server, svc domain.PriceListSvc) {
	pb.RegisterCorePriceListServiceServer(server, &priceListGRPCHandler{svc: svc})
}

func priceListToProto(l *domain.PriceList, rules *domain.PriceListRuleIDs) *pb.PriceListInfo {
	info := &pb.PriceListInfo{
		Id:          l.ID,
		Name:        l.Name,
		Description: l.Description,
		Status:      string(l.Status),
		Currency:    string(l.Currency),
		CreatedAt:   timestamppb.New(l.CreatedAt),
		UpdatedAt:   timestamppb.New(l.UpdatedAt),
	}
	if rules != nil {
		info.RuleIds = rules.IDs
		info.RulesNextCursor = rules.NextCursor
	}
	return info
}

func priceListRuleToProto(r *domain.PriceListRule) *pb.PriceListRuleInfo {
	return &pb.PriceListRuleInfo{
		Id:          r.ID,
		PriceListId: r.PriceListID,
		Scope: &pb.PriceListRuleScope{
			ItemIds:        r.Scope.ItemIDs,
			ProductLineIds: r.Scope.ProductLineIDs,
			CategoryIds:    r.Scope.CategoryIDs,
			AttributeIds:   r.Scope.AttributeIDs,
		},
		Price:     priceListPriceToProto(r.Price),
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

func priceListPriceToProto(p domain.PriceListPrice) *pb.PriceListPrice {
	out := &pb.PriceListPrice{
		Type:          string(p.Type),
		UnitPrice:     domain.UnitPricePtrToProto(p.UnitPrice),
		PercentOfBase: p.PercentOfBase,
		UnitId:        p.UnitID,
	}
	for _, t := range p.Tiers {
		out.Tiers = append(out.Tiers, &pb.PriceListTier{
			MinQuantity:   t.MinQuantity,
			PercentOfBase: t.PercentOfBase,
			UnitPrice:     domain.UnitPricePtrToProto(t.UnitPrice),
		})
	}
	return out
}

// priceListPriceFromProto validates a requested price's money (currency) and maps it; normalizeRulePrice in the service
// checks the rest.
func priceListPriceFromProto(p *pb.PriceListPrice) (domain.PriceListPrice, *apierror.APIError) {
	if p == nil {
		return domain.PriceListPrice{}, apierror.NewFieldError("price", apierror.CodeInvalidFormat, "A rule needs a price.")
	}
	unitPrice, apiErr := domain.UnitPriceFromProto("price.unit_price", p.UnitPrice)
	if apiErr != nil {
		return domain.PriceListPrice{}, apiErr
	}
	out := domain.PriceListPrice{
		Type:          constants.PriceListPriceType(p.Type),
		UnitPrice:     unitPrice,
		PercentOfBase: p.PercentOfBase,
		UnitID:        p.UnitId,
	}
	for _, t := range p.Tiers {
		tierPrice, apiErr := domain.UnitPriceFromProto("price.tiers.unit_price", t.UnitPrice)
		if apiErr != nil {
			return domain.PriceListPrice{}, apiErr
		}
		out.Tiers = append(out.Tiers, domain.PriceListTier{MinQuantity: t.MinQuantity, PercentOfBase: t.PercentOfBase, UnitPrice: tierPrice})
	}
	return out, nil
}

func grpcErr(apiErr *apierror.APIError) error {
	return contracts.ConvertAPIErrorToGRPC(apiErr)
}

func (h *priceListGRPCHandler) ListPriceLists(ctx context.Context, req *pb.ListPriceListsRequest) (*pb.ListPriceListsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	result, apiErr := h.svc.ListPriceLists(ctx, domain.ListPriceListsParams{
		Cursor:      req.Cursor,
		Limit:       req.Limit,
		Query:       req.Query,
		Statuses:    archiveStatuses(req.Statuses),
		CustomerIDs: req.CustomerIds,
	})
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	out := make([]*pb.PriceListInfo, len(result.PriceLists))
	for i, l := range result.PriceLists {
		out[i] = priceListToProto(l, nil)
	}
	return &pb.ListPriceListsResponse{PriceLists: out, PageInfo: pageInfoToProto(result.PageInfo)}, nil
}

func (h *priceListGRPCHandler) GetPriceList(ctx context.Context, req *pb.GetPriceListRequest) (*pb.PriceListInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	l, apiErr := h.svc.GetPriceList(ctx, req.Id)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	return priceListToProto(l, nil), nil
}

func (h *priceListGRPCHandler) BatchGetPriceListsByIDs(ctx context.Context, req *pb.BatchGetPriceListsByIDsRequest) (*pb.BatchGetPriceListsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	lists, rules, apiErr := h.svc.BatchGetPriceLists(ctx, req.Ids, req.WithRules)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	out := make([]*pb.PriceListInfo, len(lists))
	for i, l := range lists {
		var r *domain.PriceListRuleIDs
		if ids, ok := rules[l.ID]; ok {
			r = &ids
		}
		out[i] = priceListToProto(l, r)
	}
	return &pb.BatchGetPriceListsByIDsResponse{PriceLists: out}, nil
}

func (h *priceListGRPCHandler) CreatePriceList(ctx context.Context, req *pb.CreatePriceListRequest) (*pb.PriceListInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	l, apiErr := h.svc.CreatePriceList(ctx, domain.CreatePriceListParams{
		Name:        req.Name,
		Description: req.Description,
		Status:      archiveStatusPtr(req.Status),
		Currency:    req.Currency,
	})
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	return priceListToProto(l, nil), nil
}

func (h *priceListGRPCHandler) UpdatePriceList(ctx context.Context, req *pb.UpdatePriceListRequest) (*pb.PriceListInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePriceListParams{
		PriceListID: req.Id,
		Name:        req.Name,
		Status:      archiveStatusPtr(req.Status),
	}
	if d := req.Description; d != nil {
		if d.Clear {
			params.ClearDescription = true
		} else {
			params.Description = d.Value
		}
	}
	l, apiErr := h.svc.UpdatePriceList(ctx, params)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	return priceListToProto(l, nil), nil
}

func (h *priceListGRPCHandler) DeletePriceList(ctx context.Context, req *pb.DeletePriceListRequest) (*pb.PriceListInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	l, apiErr := h.svc.DeletePriceList(ctx, req.Id)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	return priceListToProto(l, nil), nil
}

func (h *priceListGRPCHandler) ListPriceListRules(ctx context.Context, req *pb.ListPriceListRulesRequest) (*pb.ListPriceListRulesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	result, apiErr := h.svc.ListPriceListRules(ctx, domain.ListPriceListRulesParams{
		PriceListID:    req.PriceListId,
		Cursor:         req.Cursor,
		Limit:          req.Limit,
		ItemIDs:        req.ItemIds,
		ProductLineIDs: req.ProductLineIds,
	})
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	out := make([]*pb.PriceListRuleInfo, len(result.Rules))
	for i, r := range result.Rules {
		out[i] = priceListRuleToProto(r)
	}
	return &pb.ListPriceListRulesResponse{Rules: out, PageInfo: pageInfoToProto(result.PageInfo)}, nil
}

func (h *priceListGRPCHandler) GetPriceListRule(ctx context.Context, req *pb.GetPriceListRuleRequest) (*pb.PriceListRuleInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	r, apiErr := h.svc.GetPriceListRule(ctx, req.PriceListId, req.Id)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	return priceListRuleToProto(r), nil
}

func (h *priceListGRPCHandler) BatchGetPriceListRulesByIDs(ctx context.Context, req *pb.BatchGetPriceListRulesByIDsRequest) (*pb.BatchGetPriceListRulesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	rules, apiErr := h.svc.BatchGetPriceListRules(ctx, req.Ids)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	out := make([]*pb.PriceListRuleInfo, len(rules))
	for i, r := range rules {
		out[i] = priceListRuleToProto(r)
	}
	return &pb.BatchGetPriceListRulesByIDsResponse{Rules: out}, nil
}

func (h *priceListGRPCHandler) CreatePriceListRule(ctx context.Context, req *pb.CreatePriceListRuleRequest) (*pb.PriceListRuleInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	price, apiErr := priceListPriceFromProto(req.Price)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	var scope domain.PriceListRuleScope
	if s := req.Scope; s != nil {
		scope = domain.PriceListRuleScope{ItemIDs: s.ItemIds, ProductLineIDs: s.ProductLineIds, CategoryIDs: s.CategoryIds, AttributeIDs: s.AttributeIds}
	}
	r, apiErr := h.svc.CreatePriceListRule(ctx, domain.CreatePriceListRuleParams{
		PriceListID: req.PriceListId,
		Rule:        domain.PriceListRuleInput{Scope: scope, Price: price},
	})
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	return priceListRuleToProto(r), nil
}

func (h *priceListGRPCHandler) UpdatePriceListRule(ctx context.Context, req *pb.UpdatePriceListRuleRequest) (*pb.PriceListRuleInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePriceListRuleParams{PriceListID: req.PriceListId, RuleID: req.Id}
	if req.HasItemIds {
		params.ItemIDs = &req.ItemIds
	}
	if req.HasProductLineIds {
		params.ProductLineIDs = &req.ProductLineIds
	}
	if req.HasCategoryIds {
		params.CategoryIDs = &req.CategoryIds
	}
	if req.HasAttributeIds {
		params.AttributeIDs = &req.AttributeIds
	}
	if req.Price != nil {
		price, apiErr := priceListPriceFromProto(req.Price)
		if apiErr != nil {
			return nil, grpcErr(apiErr)
		}
		params.Price = &price
	}
	r, apiErr := h.svc.UpdatePriceListRule(ctx, params)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	return priceListRuleToProto(r), nil
}

func (h *priceListGRPCHandler) DeletePriceListRule(ctx context.Context, req *pb.DeletePriceListRuleRequest) (*pb.PriceListRuleInfo, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	r, apiErr := h.svc.DeletePriceListRule(ctx, req.PriceListId, req.Id)
	if apiErr != nil {
		return nil, grpcErr(apiErr)
	}
	return priceListRuleToProto(r), nil
}

// priceListIDsIfSent is a create request's price list IDs: nil when the field wasn't sent, else never nil.
func priceListIDsIfSent(ids []string, sent bool) []string {
	if !sent {
		return nil
	}
	if ids == nil {
		return []string{}
	}
	return ids
}
