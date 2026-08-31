package accountpriceep

import (
	"context"
	"fmt"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AccountPriceSvc interface {
	ListAccountPrices(ctx context.Context, req *ListAccountPricesRequest) (*apiresource.List[apiresource.AccountPrice], *apierror.APIError)
	GetAccountPrice(ctx context.Context, req *RetrieveAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	CreateAccountPrice(ctx context.Context, req *CreateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	UpdateAccountPrice(ctx context.Context, req *UpdateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	DeleteAccountPrice(ctx context.Context, req *DeleteAccountPriceRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ExportPriceList(ctx context.Context, req *ExportPriceListRequest) (*apiresource.Job, *apierror.APIError)
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

	meta := resourcekit.GetLoadMeta(ctx)
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

	meta := resourcekit.GetLoadMeta(ctx)
	result := accountPriceFromProto(resp.AccountPrice)
	stashAccountPriceMeta(meta, resp.AccountPrice)
	return &result, nil
}

func (m *accountPriceSvcImpl) CreateAccountPrice(ctx context.Context, req *CreateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
	pbReq := &pb.CreateAccountPriceRequest{
		RecipientAccountId:    req.RecipientAccountID,
		ProductLineId:         req.ProductLineID,
		RateValue:             req.Rate.Value,
		RateNumeratorUnitId:   req.Rate.NumeratorUnitID,
		RateDenominatorUnitId: req.Rate.DenominatorUnitID,
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

	meta := resourcekit.GetLoadMeta(ctx)
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
		pbReq.RateNumeratorUnitId = &rate.NumeratorUnitID
		pbReq.RateDenominatorUnitId = &rate.DenominatorUnitID
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

	meta := resourcekit.GetLoadMeta(ctx)
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
		rate = &apiresource.Rate{
			ID:        ap.Rate.Id,
			Object:    constants.ObjectTypeRate,
			Value:     ap.Rate.Value,
			CreatedAt: grpcutil.TimestampToTime(ap.Rate.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(ap.Rate.UpdatedAt),
		}
		var numeratorAbbreviation, numeratorUnitType, denominatorAbbreviation string
		if ap.Rate.NumeratorUnit != nil {
			nu := ap.Rate.NumeratorUnit
			rate.NumeratorUnit = &apiresource.Unit{
				ID:                nu.Id,
				Object:            constants.ObjectTypeUnit,
				Name:              nu.Name,
				Abbreviation:      nu.Abbreviation,
				Type:              constants.UnitType(nu.Type),
				RatioNumerator:    nu.RatioNumerator,
				RatioDenominator:  nu.RatioDenominator,
				OffsetNumerator:   nu.OffsetNumerator,
				OffsetDenominator: nu.OffsetDenominator,
				CreatedAt:         grpcutil.TimestampToTime(nu.CreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(nu.UpdatedAt),
			}
			numeratorAbbreviation = nu.Abbreviation
			numeratorUnitType = nu.Type
		}
		if ap.Rate.DenominatorUnit != nil {
			du := ap.Rate.DenominatorUnit
			rate.DenominatorUnit = &apiresource.Unit{
				ID:                du.Id,
				Object:            constants.ObjectTypeUnit,
				Name:              du.Name,
				Abbreviation:      du.Abbreviation,
				Type:              constants.UnitType(du.Type),
				RatioNumerator:    du.RatioNumerator,
				RatioDenominator:  du.RatioDenominator,
				OffsetNumerator:   du.OffsetNumerator,
				OffsetDenominator: du.OffsetDenominator,
				CreatedAt:         grpcutil.TimestampToTime(du.CreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(du.UpdatedAt),
			}
			denominatorAbbreviation = du.Abbreviation
		}
		rate.DisplayValue = apiresource.FormatRateDisplayValue(ap.Rate.Value, numeratorAbbreviation, numeratorUnitType, denominatorAbbreviation)
	}

	return apiresource.AccountPrice{
		ID:        ap.Id,
		Object:    constants.ObjectTypeAccountPrice,
		Rate:      rate,
		CreatedAt: grpcutil.TimestampToTime(ap.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(ap.UpdatedAt),
	}
}

func stashAccountPriceMeta(meta *resourcekit.LoadMeta, ap *pb.AccountPriceInfo) {
	if ap == nil {
		return
	}

	// Stash only the recipient's account id; the customer is hydrated on demand through the shared
	// batched customer loader so the include carries the full customer (including a relationship_type
	// derived from the account's own hierarchy, not the price-list projection's stale value), not a stub.
	if ap.RecipientAccount != nil {
		meta.Set(constants.ObjectTypeAccountPrice, ap.Id, "recipient_account_id", ap.RecipientAccount.Id)
	}

	if ap.ProductLine != nil {
		pl := ap.ProductLine
		meta.Set(constants.ObjectTypeAccountPrice, ap.Id, "product_line", &apiresource.ProductLine{
			ID:               pl.Id,
			Object:           constants.ObjectTypeProductLine,
			Name:             pl.Name,
			CommissionPolicy: constants.CommissionPolicy(pl.CommissionPolicy),
			FreightPolicy:    constants.FreightPolicy(pl.FreightPolicy),
			CreatedAt:        grpcutil.TimestampToTime(pl.CreatedAt),
			UpdatedAt:        grpcutil.TimestampToTime(pl.UpdatedAt),
		})
	}

	categories := make([]apiresource.ItemCategory, len(ap.Categories))
	for i, c := range ap.Categories {
		categories[i] = apiresource.ItemCategory{
			ID:        c.Id,
			Object:    constants.ObjectTypeItemCategory,
			Name:      c.Name,
			Type:      constants.ItemCategoryType(c.Type),
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
			CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
		}
	}
	meta.Set(constants.ObjectTypeAccountPrice, ap.Id, "attributes", apiresource.NewList(attributes, apiresource.PageInfo{}))
}

// ExportPriceList accepts the export and returns the job that tracks it. The document is built by the export worker, which prices the whole catalog against one pricing bundle rather than a request's worth of round trips.
func (m *accountPriceSvcImpl) ExportPriceList(ctx context.Context, req *ExportPriceListRequest) (*apiresource.Job, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.export_price_list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportPriceListResponse, error) {
			return m.coreClient.ExportPriceList(ctx, &pb.ExportPriceListRequest{CustomerAccountId: req.CustomerID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(resp.GetJob()), nil
}
