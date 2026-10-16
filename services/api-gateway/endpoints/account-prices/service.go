package accountpriceep

import (
	"context"
	"fmt"

	"github.com/open-mrp/apikit/include"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AccountPriceSvc interface {
	ListAccountPrices(ctx context.Context, req *ListAccountPricesRequest) (*apiresource.List[apiresource.AccountPrice], *apierror.APIError)
	GetAccountPrice(ctx context.Context, req *RetrieveAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	CreateAccountPrice(ctx context.Context, req *CreateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	UpdateAccountPrice(ctx context.Context, req *UpdateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	DeleteAccountPrice(ctx context.Context, req *DeleteAccountPriceRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ExportPriceList(ctx context.Context, req *ExportPriceListRequest) (*apiresource.AsyncJob, *apierror.APIError)
}

type AccountPriceSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
	// SalesClient (required) is the core-service sales gRPC client, used by the price
	// list export to quote prices through the same engine that prices sales orders.
	SalesClient pb.CoreSalesServiceClient
}

type accountPriceSvcImpl struct {
	coreClient  pb.CoreServiceClient
	salesClient pb.CoreSalesServiceClient
}

var accountPriceSvcTracer = tracing.GetTracer("api-gateway.endpoints.account_prices.service")

func (c *AccountPriceSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account price endpoint service: core client is required")
	}
	if c.SalesClient == nil {
		return fmt.Errorf("account price endpoint service: sales client is required")
	}
	return nil
}

func NewAccountPriceSvc(config *AccountPriceSvcConfig) AccountPriceSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountPriceSvcImpl{
		coreClient:  config.CoreClient,
		salesClient: config.SalesClient,
	}
}

func (m *accountPriceSvcImpl) ListAccountPrices(ctx context.Context, req *ListAccountPricesRequest) (*apiresource.List[apiresource.AccountPrice], *apierror.APIError) {
	pbReq := &pb.ListAccountPricesRequest{
		Cursor:             req.Cursor,
		Limit:              req.Limit,
		Query:              req.Query,
		RecipientAccountId: req.RecipientAccountID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountPricesResponse, error) {
			return m.coreClient.ListAccountPrices(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := include.GetLoadMeta(ctx)
	prices := make([]apiresource.AccountPrice, len(resp.AccountPrices))
	for i, ap := range resp.AccountPrices {
		prices[i] = accountPriceFromProto(ap)
		stashAccountPriceMeta(meta, ap)
	}

	return apiresource.NewList(prices, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *accountPriceSvcImpl) GetAccountPrice(ctx context.Context, req *RetrieveAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
	pbReq := &pb.GetAccountPriceRequest{
		Id: req.AccountPriceID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountPriceResponse, error) {
			return m.coreClient.GetAccountPrice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := include.GetLoadMeta(ctx)
	result := accountPriceFromProto(resp.AccountPrice)
	stashAccountPriceMeta(meta, resp.AccountPrice)
	return &result, nil
}

func (m *accountPriceSvcImpl) CreateAccountPrice(ctx context.Context, req *CreateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
	pbReq := &pb.CreateAccountPriceRequest{
		RecipientAccountId:    req.RecipientAccountID,
		ProductLineId:         req.ProductLineID,
		RateValue:             req.Rate.Value,
		RateNumeratorUnitId:   req.Rate.UnitID,
		RateDenominatorUnitId: req.Rate.PerUnitID,
		CategoryIds:           req.CategoryIDs,
		AttributeIds:          req.AttributeIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAccountPriceResponse, error) {
			return m.coreClient.CreateAccountPrice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := include.GetLoadMeta(ctx)
	result := accountPriceFromProto(resp.AccountPrice)
	stashAccountPriceMeta(meta, resp.AccountPrice)
	return &result, nil
}

func (m *accountPriceSvcImpl) UpdateAccountPrice(ctx context.Context, req *UpdateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
	pbReq := &pb.UpdateAccountPriceRequest{
		Id:                 req.AccountPriceID,
		RecipientAccountId: req.RecipientAccountID.Ptr(),
		ProductLineId:      req.ProductLineID.Ptr(),
	}

	// The rate is replaced whole, so its three columns move together or not at all.
	if rate, ok := req.Rate.Value(); ok {
		pbReq.RateValue = &rate.Value
		pbReq.RateNumeratorUnitId = &rate.UnitID
		pbReq.RateDenominatorUnitId = &rate.PerUnitID
	}

	if v, ok := req.CategoryIDs.Value(); ok {
		pbReq.CategoryIds = &pb.AccountPriceIDList{Ids: v}
	}
	if v, ok := req.AttributeIDs.Value(); ok {
		pbReq.AttributeIds = &pb.AccountPriceIDList{Ids: v}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountPriceResponse, error) {
			return m.coreClient.UpdateAccountPrice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := include.GetLoadMeta(ctx)
	result := accountPriceFromProto(resp.AccountPrice)
	stashAccountPriceMeta(meta, resp.AccountPrice)
	return &result, nil
}

func (m *accountPriceSvcImpl) DeleteAccountPrice(ctx context.Context, req *DeleteAccountPriceRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteAccountPriceRequest{
		Id: req.AccountPriceID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteAccountPrice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func accountPriceFromProto(ap *pb.AccountPriceInfo) apiresource.AccountPrice {
	if ap == nil {
		return apiresource.AccountPrice{}
	}

	var rate *apiresource.Rate
	if ap.Rate != nil {
		rate = apiresource.NewRate(ap.Rate.Value, ap.Rate.GetNumeratorUnit().GetId(), ap.Rate.GetDenominatorUnit().GetId())
	}

	return apiresource.AccountPrice{
		ID:        ap.Id,
		Object:    constants.ObjectTypeAccountPrice,
		Rate:      rate,
		CreatedAt: grpcutil.TimestampToTime(ap.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ap.UpdatedAt),
	}
}

func stashAccountPriceMeta(meta *include.LoadMeta, ap *pb.AccountPriceInfo) {
	if ap == nil {
		return
	}

	// Stash only the recipient's account id; the customer is hydrated on demand through the shared
	// batched customer loader so the include carries the full customer (including a relationship_type
	// derived from the account's own hierarchy, not the price-list projection's stale value), not a stub.
	if ap.RecipientAccount != nil {
		meta.Set(constants.ObjectTypeAccountPrice, ap.Id, "recipient_account_id", ap.RecipientAccount.Id)
	}

	// The product line is hydrated through the shared loader, so the include is the full product line.
	if ap.ProductLine != nil {
		meta.Set(constants.ObjectTypeAccountPrice, ap.Id, "product_line_id", ap.ProductLine.Id)
	}

	categories := make([]apiresource.ItemCategory, len(ap.Categories))
	for i, c := range ap.Categories {
		categories[i] = apiresource.ItemCategory{
			ID:        c.Id,
			Object:    constants.ObjectTypeItemCategory,
			Name:      c.Name,
			Type:      constants.ItemCategoryTypeFromCode(c.Type),
			CreatedAt: grpcutil.TimestampToTime(c.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(c.UpdatedAt),
		}
	}
	meta.Set(constants.ObjectTypeAccountPrice, ap.Id, "categories", apiresource.NewList(categories, apiresource.PageInfo{}))

	attributes := make([]apiresource.Attribute, len(ap.Attributes))
	for i, a := range ap.Attributes {
		attributes[i] = apiresource.Attribute{
			ID:        a.Id,
			Object:    constants.ObjectTypeAttribute,
			Value:     a.Value,
			ColorCode: constants.Color(a.ColorCode),
			Status:    constants.ArchiveStatus(a.Status),
			CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
		}
	}
	meta.Set(constants.ObjectTypeAccountPrice, ap.Id, "attributes", apiresource.NewList(attributes, apiresource.PageInfo{}))
}

// ExportPriceList accepts the export and returns the job that tracks it. The document is built by the export worker, which prices the whole catalog against one pricing bundle rather than a request's worth of round trips.
func (m *accountPriceSvcImpl) ExportPriceList(ctx context.Context, req *ExportPriceListRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.export_price_list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportPriceListResponse, error) {
			return m.coreClient.ExportPriceList(ctx, &pb.ExportPriceListRequest{CustomerAccountId: req.CustomerID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}
