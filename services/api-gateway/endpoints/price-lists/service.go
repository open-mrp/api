package pricelistep

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

const (
	priceListsRoute = "/v1/sales/price-lists"
	priceListRoute  = priceListsRoute + "/{id}"
	rulesRoute      = priceListsRoute + "/{price_list_id}/rules"
	ruleRoute       = rulesRoute + "/{id}"
)

// priceListIncludes are the include keys a price list response accepts.
var priceListIncludes = []string{"rules", "rules.scope.items", "rules.scope.product_lines", "rules.scope.categories", "rules.scope.attributes", "rules.scope.attributes.property"}

// ruleIncludes are the include keys a price-list rule response accepts.
var ruleIncludes = []string{"scope.items", "scope.product_lines", "scope.categories", "scope.attributes", "scope.attributes.property"}

type PriceListSvc interface {
	ListPriceLists(ctx context.Context, req *ListPriceListsRequest) (*apiresource.List[apiresource.PriceList], *apierror.APIError)
	GetPriceList(ctx context.Context, req *RetrievePriceListRequest) (*apiresource.PriceList, *apierror.APIError)
	CreatePriceList(ctx context.Context, req *CreatePriceListRequest) (*apiresource.PriceList, *apierror.APIError)
	UpdatePriceList(ctx context.Context, req *UpdatePriceListRequest) (*apiresource.PriceList, *apierror.APIError)
	DeletePriceList(ctx context.Context, req *DeletePriceListRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ListRules(ctx context.Context, req *ListPriceListRulesRequest) (*apiresource.List[apiresource.PriceListRule], *apierror.APIError)
	GetRule(ctx context.Context, req *RetrievePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError)
	CreateRule(ctx context.Context, req *CreatePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError)
	UpdateRule(ctx context.Context, req *UpdatePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError)
	DeleteRule(ctx context.Context, req *DeletePriceListRuleRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ExportCustomerPrices(ctx context.Context, req *ExportCustomerPricesRequest) (*apiresource.AsyncJob, *apierror.APIError)
}

type PriceListSvcConfig struct {
	// CoreClient (required) is the core-service price list gRPC client.
	CoreClient pb.CorePriceListServiceClient
	// ExportClient (required) is the core-service client that accepts exports.
	ExportClient pb.CoreServiceClient
}

func (c *PriceListSvcConfig) validate() error {
	if c.CoreClient == nil || c.ExportClient == nil {
		return fmt.Errorf("price list endpoint service: core and export clients are required")
	}
	return nil
}

type priceListSvcImpl struct {
	coreClient   pb.CorePriceListServiceClient
	exportClient pb.CoreServiceClient
}

var priceListSvcTracer = tracing.GetTracer("api-gateway.endpoints.price-lists.service")

func NewPriceListSvc(config *PriceListSvcConfig) PriceListSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &priceListSvcImpl{coreClient: config.CoreClient, exportClient: config.ExportClient}
}

func call[T any](ctx context.Context, name string, fn func(context.Context, ...grpc.CallOption) (T, error)) (T, *apierror.APIError) {
	return grpcutil.CallRPC(ctx, priceListSvcTracer, name, domain.ServiceName, fn)
}

func enumStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func loadList(ctx context.Context, id string) (*apiresource.PriceList, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadPriceLists(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewNotFoundError("Price list not found.")
	}
	return v.(*apiresource.PriceList), nil
}

func (s *priceListSvcImpl) ListPriceLists(ctx context.Context, req *ListPriceListsRequest) (*apiresource.List[apiresource.PriceList], *apierror.APIError) {
	resp, apiErr := call(ctx, "service.price_lists.list", func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPriceListsResponse, error) {
		return s.coreClient.ListPriceLists(ctx, &pb.ListPriceListsRequest{
			Cursor: req.Cursor, Limit: req.Limit, Query: req.Query, Statuses: enumStrings(req.Statuses), CustomerIds: req.CustomerIDs,
		}, opts...)
	})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.PriceLists))
	for i, l := range resp.PriceLists {
		ids[i] = l.Id
	}
	loaded, apiErr := resourceloaders.LoadPriceLists(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.PriceList, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.PriceList)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (s *priceListSvcImpl) GetPriceList(ctx context.Context, req *RetrievePriceListRequest) (*apiresource.PriceList, *apierror.APIError) {
	if _, apiErr := call(ctx, "service.price_lists.get", func(ctx context.Context, opts ...grpc.CallOption) (*pb.PriceListInfo, error) {
		return s.coreClient.GetPriceList(ctx, &pb.GetPriceListRequest{Id: req.ID}, opts...)
	}); apiErr != nil {
		return nil, apiErr
	}
	return loadList(ctx, req.ID)
}

func (s *priceListSvcImpl) CreatePriceList(ctx context.Context, req *CreatePriceListRequest) (*apiresource.PriceList, *apierror.APIError) {
	resp, apiErr := call(ctx, "service.price_lists.create", func(ctx context.Context, opts ...grpc.CallOption) (*pb.PriceListInfo, error) {
		return s.coreClient.CreatePriceList(ctx, &pb.CreatePriceListRequest{
			Name:        req.Name,
			Description: req.Description.Ptr(),
			Status:      req.Status.Ptr().StringPtr(),
			Currency:    req.Currency.Ptr(),
		}, opts...)
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadList(ctx, resp.Id)
}

func (s *priceListSvcImpl) UpdatePriceList(ctx context.Context, req *UpdatePriceListRequest) (*apiresource.PriceList, *apierror.APIError) {
	resp, apiErr := call(ctx, "service.price_lists.update", func(ctx context.Context, opts ...grpc.CallOption) (*pb.PriceListInfo, error) {
		return s.coreClient.UpdatePriceList(ctx, &pb.UpdatePriceListRequest{
			Id:          req.ID,
			Name:        req.Name.Ptr(),
			Description: fieldpb.StringClearableToProto(req.Description),
			Status:      req.Status.Ptr().StringPtr(),
		}, opts...)
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadList(ctx, resp.Id)
}

func (s *priceListSvcImpl) DeletePriceList(ctx context.Context, req *DeletePriceListRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	if _, apiErr := call(ctx, "service.price_lists.delete", func(ctx context.Context, opts ...grpc.CallOption) (*pb.PriceListInfo, error) {
		return s.coreClient.DeletePriceList(ctx, &pb.DeletePriceListRequest{Id: req.ID}, opts...)
	}); apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// presentRules is core's rules as the response shows them, scopes ready for their includes.
func presentRules(ctx context.Context, rules []*pb.PriceListRuleInfo) []apiresource.PriceListRule {
	out := make([]apiresource.PriceListRule, len(rules))
	for i, r := range rules {
		out[i] = *resourceloaders.PriceListRuleFromProto(ctx, r)
	}
	return out
}

func (s *priceListSvcImpl) ListRules(ctx context.Context, req *ListPriceListRulesRequest) (*apiresource.List[apiresource.PriceListRule], *apierror.APIError) {
	resp, apiErr := call(ctx, "service.price_lists.list_rules", func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPriceListRulesResponse, error) {
		return s.coreClient.ListPriceListRules(ctx, &pb.ListPriceListRulesRequest{
			PriceListId: req.PriceListID, Cursor: req.Cursor, Limit: req.Limit, ItemIds: req.ItemIDs, ProductLineIds: req.ProductLineIDs,
		}, opts...)
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return apiresource.NewList(presentRules(ctx, resp.Rules), grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (s *priceListSvcImpl) GetRule(ctx context.Context, req *RetrievePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError) {
	resp, apiErr := call(ctx, "service.price_lists.get_rule", func(ctx context.Context, opts ...grpc.CallOption) (*pb.PriceListRuleInfo, error) {
		return s.coreClient.GetPriceListRule(ctx, &pb.GetPriceListRuleRequest{PriceListId: req.PriceListID, Id: req.ID}, opts...)
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.PriceListRuleFromProto(ctx, resp), nil
}

func (s *priceListSvcImpl) CreateRule(ctx context.Context, req *CreatePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError) {
	pbReq := &pb.CreatePriceListRuleRequest{PriceListId: req.PriceListID, Price: req.Price.toProto(), Scope: &pb.PriceListRuleScope{}}
	if scope, ok := req.Scope.Value(); ok {
		pbReq.Scope = &pb.PriceListRuleScope{
			ItemIds: scope.ItemIDs, ProductLineIds: scope.ProductLineIDs, CategoryIds: scope.CategoryIDs, AttributeIds: scope.AttributeIDs,
		}
	}
	resp, apiErr := call(ctx, "service.price_lists.create_rule", func(ctx context.Context, opts ...grpc.CallOption) (*pb.PriceListRuleInfo, error) {
		return s.coreClient.CreatePriceListRule(ctx, pbReq, opts...)
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.PriceListRuleFromProto(ctx, resp), nil
}

func (s *priceListSvcImpl) UpdateRule(ctx context.Context, req *UpdatePriceListRuleRequest) (*apiresource.PriceListRule, *apierror.APIError) {
	pbReq := &pb.UpdatePriceListRuleRequest{PriceListId: req.PriceListID, Id: req.ID}
	if scope, ok := req.Scope.Value(); ok {
		if v, ok := scope.ItemIDs.Value(); ok {
			pbReq.ItemIds, pbReq.HasItemIds = v, true
		}
		if v, ok := scope.ProductLineIDs.Value(); ok {
			pbReq.ProductLineIds, pbReq.HasProductLineIds = v, true
		}
		if v, ok := scope.CategoryIDs.Value(); ok {
			pbReq.CategoryIds, pbReq.HasCategoryIds = v, true
		}
		if v, ok := scope.AttributeIDs.Value(); ok {
			pbReq.AttributeIds, pbReq.HasAttributeIds = v, true
		}
	}
	if price, ok := req.Price.Value(); ok {
		pbReq.Price = price.toProto()
	}
	resp, apiErr := call(ctx, "service.price_lists.update_rule", func(ctx context.Context, opts ...grpc.CallOption) (*pb.PriceListRuleInfo, error) {
		return s.coreClient.UpdatePriceListRule(ctx, pbReq, opts...)
	})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.PriceListRuleFromProto(ctx, resp), nil
}

func (s *priceListSvcImpl) DeleteRule(ctx context.Context, req *DeletePriceListRuleRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	if _, apiErr := call(ctx, "service.price_lists.delete_rule", func(ctx context.Context, opts ...grpc.CallOption) (*pb.PriceListRuleInfo, error) {
		return s.coreClient.DeletePriceListRule(ctx, &pb.DeletePriceListRuleRequest{PriceListId: req.PriceListID, Id: req.ID}, opts...)
	}); apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

func (p PriceListPriceInput) toProto() *pb.PriceListPrice {
	out := &pb.PriceListPrice{
		Type:          string(p.Type),
		UnitPrice:     apirequest.OptionalUnitPriceToProto(p.UnitPrice),
		PercentOfBase: p.PercentOfBase.Ptr(),
	}
	if v, ok := p.UnitID.Value(); ok {
		out.UnitId = v
	}
	for _, t := range p.Tiers {
		out.Tiers = append(out.Tiers, &pb.PriceListTier{
			MinQuantity:   t.MinQuantity,
			PercentOfBase: t.PercentOfBase.Ptr(),
			UnitPrice:     apirequest.OptionalUnitPriceToProto(t.UnitPrice),
		})
	}
	return out
}

// sampleRuleScope is the docs example scope.
var sampleRuleScope = PriceListRuleScopeInput{ProductLineIDs: []string{apiresource.SampleProductLineID}}
