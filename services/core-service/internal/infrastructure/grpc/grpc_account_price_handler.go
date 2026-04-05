package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func accountPriceToProto(ap *domain.AccountPrice) *pb.AccountPriceInfo {
	cats := make([]*pb.AccountPriceCategoryInfo, len(ap.Categories))
	for i, c := range ap.Categories {
		cats[i] = &pb.AccountPriceCategoryInfo{
			Id:   c.ID,
			Name: c.Name,
		}
	}

	attrs := make([]*pb.AccountPriceAttributeInfo, len(ap.Attributes))
	for i, a := range ap.Attributes {
		attrs[i] = &pb.AccountPriceAttributeInfo{
			Id:    a.ID,
			Value: a.Value,
		}
	}

	return &pb.AccountPriceInfo{
		Id: ap.ID,
		RecipientAccount: &pb.AccountPriceRecipientInfo{
			Id:   ap.RecipientAccountID,
			Name: ap.RecipientAccountName,
		},
		ProductLine: &pb.AccountPriceProductLineInfo{
			Id:   ap.ProductLineID,
			Name: ap.ProductLineName,
		},
		Rate: &pb.AccountPriceRateInfo{
			Id:    ap.RateID,
			Value: ap.RateValue,
			NumeratorUnit: &pb.AccountPriceUnitInfo{
				Id:           ap.NumeratorUnitID,
				Name:         ap.NumeratorUnitName,
				Abbreviation: ap.NumeratorUnitAbbr,
				Type:         ap.NumeratorUnitType,
			},
			DenominatorUnit: &pb.AccountPriceUnitInfo{
				Id:           ap.DenominatorUnitID,
				Name:         ap.DenominatorUnitName,
				Abbreviation: ap.DenominatorUnitAbbr,
				Type:         ap.DenominatorUnitType,
			},
		},
		Categories: cats,
		Attributes: attrs,
		CreatedAt:  timestamppb.New(ap.CreatedAt),
		UpdatedAt:  timestamppb.New(ap.UpdatedAt),
	}
}

func (h *gRPCHandler) ListAccountPrices(ctx context.Context, req *pb.ListAccountPricesRequest) (*pb.ListAccountPricesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAccountPricesParams{
		Cursor:             req.Cursor,
		Limit:              req.Limit,
		Query:              req.Query,
		RecipientAccountID: req.RecipientAccountId,
	}

	result, apiErr := h.accountPriceSvc.ListAccountPrices(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbPrices := make([]*pb.AccountPriceInfo, len(result.AccountPrices))
	for i, ap := range result.AccountPrices {
		pbPrices[i] = accountPriceToProto(ap)
	}

	return &pb.ListAccountPricesResponse{
		AccountPrices: pbPrices,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetAccountPrice(ctx context.Context, req *pb.GetAccountPriceRequest) (*pb.GetAccountPriceResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ap, apiErr := h.accountPriceSvc.GetAccountPrice(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountPriceResponse{
		AccountPrice: accountPriceToProto(ap),
	}, nil
}

func (h *gRPCHandler) CreateAccountPrice(ctx context.Context, req *pb.CreateAccountPriceRequest) (*pb.CreateAccountPriceResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateAccountPriceParams{
		RecipientAccountID:    req.RecipientAccountId,
		ProductLineID:         req.ProductLineId,
		RateValue:             req.RateValue,
		RateNumeratorUnitID:   req.RateNumeratorUnitId,
		RateDenominatorUnitID: req.RateDenominatorUnitId,
		CategoryIDs:           req.CategoryIds,
		AttributeIDs:          req.AttributeIds,
	}

	ap, apiErr := h.accountPriceSvc.CreateAccountPrice(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAccountPriceResponse{
		AccountPrice: accountPriceToProto(ap),
	}, nil
}

func (h *gRPCHandler) UpdateAccountPrice(ctx context.Context, req *pb.UpdateAccountPriceRequest) (*pb.UpdateAccountPriceResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateAccountPriceParams{
		AccountPriceID:        req.Id,
		RecipientAccountID:    req.RecipientAccountId,
		ProductLineID:         req.ProductLineId,
		RateValue:             req.RateValue,
		RateNumeratorUnitID:   req.RateNumeratorUnitId,
		RateDenominatorUnitID: req.RateDenominatorUnitId,
	}

	if req.CategoryIds != nil {
		ids := req.CategoryIds.Ids
		params.CategoryIDs = &ids
	}
	if req.AttributeIds != nil {
		ids := req.AttributeIds.Ids
		params.AttributeIDs = &ids
	}

	ap, apiErr := h.accountPriceSvc.UpdateAccountPrice(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAccountPriceResponse{
		AccountPrice: accountPriceToProto(ap),
	}, nil
}

func (h *gRPCHandler) DeleteAccountPrice(ctx context.Context, req *pb.DeleteAccountPriceRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountPriceSvc.DeleteAccountPrice(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
