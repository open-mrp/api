package grpc

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
)

func (h *gRPCHandler) SetItemOrderUnits(ctx context.Context, req *pb.SetItemOrderUnitsRequest) (*pb.SetItemOrderUnitsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	orderUnits, apiErr := orderUnitsFromProto("", req.OrderUnits)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	job, apiErr := h.itemSvc.SetItemOrderUnits(ctx, domain.SetOrderUnitsParams{
		Filter: domain.SetOrderUnitsFilter{
			ItemIDs:        req.ItemIds,
			ProductLineIDs: req.ProductLineIds,
			CategoryIDs:    req.CategoryIds,
			AttributeIDs:   req.AttributeIds,
		},
		OrderUnits:         orderUnits,
		DefaultOrderUnitID: req.DefaultOrderUnitId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.SetItemOrderUnitsResponse{Job: jobToProto(job)}, nil
}

func (h *gRPCHandler) ChangeItemStockingUnit(ctx context.Context, req *pb.ChangeItemStockingUnitRequest) (*pb.ChangeItemStockingUnitResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	job, apiErr := h.itemSvc.ChangeItemStockingUnit(ctx, domain.ChangeStockingUnitParams{
		ItemID:         req.Id,
		StockingUnitID: req.StockingUnitId,
		Mode:           domain.ChangeStockingUnitMode(req.Mode),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.ChangeItemStockingUnitResponse{Job: jobToProto(job)}, nil
}

// orderUnitsFromProto converts requested order units, validating each discount; prefix is the request path they sit
// under.
func orderUnitsFromProto(prefix string, in []*pb.ItemOrderUnitInfo) ([]domain.ItemOrderUnit, *apierror.APIError) {
	out := make([]domain.ItemOrderUnit, 0, len(in))
	for i, ou := range in {
		if ou == nil {
			continue
		}
		discount, apiErr := domain.DiscountFromProto(fmt.Sprintf("%sorder_units[%d].discount", prefix, i), ou.Discount)
		if apiErr != nil {
			return nil, apiErr
		}
		out = append(out, domain.ItemOrderUnit{
			UnitID:           ou.UnitId,
			PortalVisibility: constants.CustomerPortalVisibility(ou.PortalVisibility),
			Discount:         discount,
			StockingQuantity: ou.Quantity,
		})
	}
	return out, nil
}

func orderUnitsToProto(in []domain.ItemOrderUnit) []*pb.ItemOrderUnitInfo {
	out := make([]*pb.ItemOrderUnitInfo, len(in))
	for i, ou := range in {
		info := &pb.ItemOrderUnitInfo{
			UnitId:           ou.UnitID,
			PortalVisibility: string(ou.PortalVisibility),
			Quantity:         ou.StockingQuantity,
		}
		if ou.Discount != nil {
			info.Discount = ou.Discount.ToProto()
		}
		out[i] = info
	}
	return out
}
