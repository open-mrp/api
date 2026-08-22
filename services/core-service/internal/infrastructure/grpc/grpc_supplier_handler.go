package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func supplierSummaryToProto(s *domain.SupplierSummary) *pb.SupplierSummaryProto {
	return &pb.SupplierSummaryProto{
		Id:            s.ID,
		Name:          s.Name,
		Number:        s.Number,
		MaterialCount: s.MaterialCount,
		CreatedAt:     timestamppb.New(s.CreatedAt),
	}
}

func supplierToProto(s *domain.Supplier) *pb.SupplierProto {
	p := &pb.SupplierProto{
		Id:            s.ID,
		Name:          s.Name,
		Number:        s.Number,
		Note:          s.Note,
		MaterialCount: s.MaterialCount,
		CreatedAt:     timestamppb.New(s.CreatedAt),
		UpdatedAt:     timestamppb.New(s.UpdatedAt),
	}

	if s.BillToAddress != nil {
		p.BillToAddress = customerAddressToProto(s.BillToAddress)
	}

	if s.ShipToAddress != nil {
		p.ShipToAddress = customerAddressToProto(s.ShipToAddress)
	}

	return p
}

func (h *gRPCHandler) ListSuppliers(ctx context.Context, req *pb.ListSuppliersRequest) (*pb.ListSuppliersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListSuppliersParams{
		Limit: req.Limit,
	}

	if req.Cursor != nil {
		params.Cursor = req.Cursor
	}
	if req.Query != nil {
		params.Query = req.Query
	}
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	params.ItemIDs = req.ItemIds

	result, apiErr := h.supplierSvc.ListSuppliers(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	suppliers := make([]*pb.SupplierSummaryProto, len(result.Items))
	for i, s := range result.Items {
		suppliers[i] = supplierSummaryToProto(s)
	}

	return &pb.ListSuppliersResponse{
		Suppliers: suppliers,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetSupplier(ctx context.Context, req *pb.GetSupplierRequest) (*pb.GetSupplierResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	supplier, apiErr := h.supplierSvc.GetSupplier(ctx, domain.GetSupplierParams{
		SupplierID: req.Id,
		Includes:   req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSupplierResponse{
		Supplier: supplierToProto(supplier),
	}, nil
}

func (h *gRPCHandler) CreateSupplier(ctx context.Context, req *pb.CreateSupplierRequest) (*pb.CreateSupplierResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateSupplierParams{
		Name:     req.Name,
		Number:   req.Number,
		Note:     req.Note,
		Includes: req.Includes,
	}

	if req.BillToAddress != nil {
		params.BillToAddress = protoAddressInputToCreateParams(req.BillToAddress)
	}

	if req.ShipToAddress != nil {
		params.ShipToAddress = protoAddressInputToCreateParams(req.ShipToAddress)
	}

	supplier, apiErr := h.supplierSvc.CreateSupplier(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSupplierResponse{
		Supplier: supplierToProto(supplier),
	}, nil
}

func (h *gRPCHandler) UpdateSupplier(ctx context.Context, req *pb.UpdateSupplierRequest) (*pb.UpdateSupplierResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateSupplierParams{
		SupplierID:      req.Id,
		Name:            req.Name,
		Number:          req.Number,
		Note:            req.Note,
		UpdateNote:      req.UpdateNote,
		BillToAddressID: req.BillToAddressId,
		ShipToAddressID: req.ShipToAddressId,
		Includes:        req.Includes,
	}

	supplier, apiErr := h.supplierSvc.UpdateSupplier(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateSupplierResponse{
		Supplier: supplierToProto(supplier),
	}, nil
}

func (h *gRPCHandler) DeleteSupplier(ctx context.Context, req *pb.DeleteSupplierRequest) (*pb.DeleteSupplierResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	supplier, apiErr := h.supplierSvc.DeleteSupplier(ctx, domain.DeleteSupplierParams{
		SupplierID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteSupplierResponse{
		Supplier: supplierToProto(supplier),
	}, nil
}

func (h *gRPCHandler) BulkDeleteSuppliers(ctx context.Context, req *pb.BulkDeleteSuppliersRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.supplierSvc.BulkDeleteSuppliers(ctx, domain.BulkDeleteSuppliersParams{
		SupplierIDs: req.SupplierIds,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func protoAddressInputToCreateParams(input *pb.CreateSupplierAddressInput) *domain.CreateAddressParams {
	if input == nil {
		return nil
	}

	return &domain.CreateAddressParams{
		Name:        input.Name,
		Phone:       input.Phone,
		Email:       input.Email,
		IsDropShip:  input.IsDropShip,
		StreetLine1: input.StreetLine_1,
		StreetLine2: input.StreetLine_2,
		Locality:    input.Locality,
		State:       input.State,
		PostalCode:  input.PostalCode,
		Country:     input.Country,
	}
}
