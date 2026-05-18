package accountpriceep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AccountPriceSvc interface {
	ListAccountPrices(ctx context.Context, req *ListAccountPricesRequest) (*apiresource.List[apiresource.AccountPrice], *apierror.APIError)
	GetAccountPrice(ctx context.Context, req *RetrieveAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	CreateAccountPrice(ctx context.Context, req *CreateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	UpdateAccountPrice(ctx context.Context, req *UpdateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError)
	DeleteAccountPrice(ctx context.Context, req *DeleteAccountPriceRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AccountPriceSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type accountPriceSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var accountPriceSvcTracer = tracing.GetTracer("api-gateway.endpoints.account_prices.service")

func (c *AccountPriceSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account price endpoint service: core client is required")
	}
	return nil
}

func NewAccountPriceSvc(config *AccountPriceSvcConfig) AccountPriceSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountPriceSvcImpl{
		coreClient: config.CoreClient,
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

	return AccountPriceListPresenter(ctx, resp), nil
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

	result := AccountPricePresenter(resp.AccountPrice)
	return &result, nil
}

func (m *accountPriceSvcImpl) CreateAccountPrice(ctx context.Context, req *CreateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
	pbReq := &pb.CreateAccountPriceRequest{
		RecipientAccountId:    req.RecipientAccountID,
		ProductLineId:         req.ProductLineID,
		RateValue:             req.RateValue,
		RateNumeratorUnitId:   req.RateNumeratorUnitID,
		RateDenominatorUnitId: req.RateDenominatorUnitID,
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

	result := AccountPricePresenter(resp.AccountPrice)
	return &result, nil
}

func (m *accountPriceSvcImpl) UpdateAccountPrice(ctx context.Context, req *UpdateAccountPriceRequest) (*apiresource.AccountPrice, *apierror.APIError) {
	pbReq := &pb.UpdateAccountPriceRequest{
		Id:                    req.AccountPriceID,
		RecipientAccountId:    req.RecipientAccountID,
		ProductLineId:         req.ProductLineID,
		RateValue:             req.RateValue,
		RateNumeratorUnitId:   req.RateNumeratorUnitID,
		RateDenominatorUnitId: req.RateDenominatorUnitID,
	}

	if req.CategoryIDs != nil {
		pbReq.CategoryIds = &pb.AccountPriceIDList{Ids: *req.CategoryIDs}
	}
	if req.AttributeIDs != nil {
		pbReq.AttributeIds = &pb.AccountPriceIDList{Ids: *req.AttributeIDs}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountPriceResponse, error) {
			return m.coreClient.UpdateAccountPrice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountPricePresenter(resp.AccountPrice)
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
