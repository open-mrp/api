package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/field"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func carrierBillingTypePtr(t *constants.CarrierBillingType) *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}

func supplierSummaryToProto(s *domain.Supplier) *pb.SupplierSummaryProto {
	p := &pb.SupplierSummaryProto{
		Id:                    s.ID,
		Name:                  s.Name,
		Number:                s.Number,
		BillToAddressId:       s.BillToAddressID,
		ShipToAddressId:       s.ShipToAddressID,
		Status:                string(s.ArchiveStatus),
		Instructions:          s.Instructions,
		Email:                 s.Email,
		Phone:                 s.Phone,
		Url:                   s.URL,
		CarrierBillingType:    carrierBillingTypePtr(s.CarrierBillingType),
		CarrierBillingAccount: s.CarrierBillingAccount,
		DefaultCarrierId:      s.DefaultCarrierID,
		DefaultServiceLevelId: s.DefaultServiceLevelID,
		PaymentTermId:         s.PaymentTermID,
		ShippingTermId:        s.ShippingTermID,
		CreatedAt:             timestamppb.New(s.CreatedAt),
		UpdatedAt:             timestamppb.New(s.UpdatedAt),
	}
	if s.MaterialCount != nil {
		p.MaterialCount, p.HasMaterialCount = *s.MaterialCount, true
	}
	if s.BillToAddress != nil {
		p.BillToAddress = customerAddressToProto(s.BillToAddress)
	}
	if s.ShipToAddress != nil {
		p.ShipToAddress = customerAddressToProto(s.ShipToAddress)
	}
	return p
}

func supplierToProto(s *domain.Supplier) *pb.SupplierProto {
	p := &pb.SupplierProto{
		Id:                    s.ID,
		Name:                  s.Name,
		Number:                s.Number,
		BillToAddressId:       s.BillToAddressID,
		ShipToAddressId:       s.ShipToAddressID,
		Status:                string(s.ArchiveStatus),
		Instructions:          s.Instructions,
		Email:                 s.Email,
		Phone:                 s.Phone,
		Url:                   s.URL,
		CarrierBillingType:    carrierBillingTypePtr(s.CarrierBillingType),
		CarrierBillingAccount: s.CarrierBillingAccount,
		DefaultCarrierId:      s.DefaultCarrierID,
		DefaultServiceLevelId: s.DefaultServiceLevelID,
		PaymentTermId:         s.PaymentTermID,
		ShippingTermId:        s.ShippingTermID,
		CreatedAt:             timestamppb.New(s.CreatedAt),
		UpdatedAt:             timestamppb.New(s.UpdatedAt),
	}
	if s.MaterialCount != nil {
		p.MaterialCount, p.HasMaterialCount = *s.MaterialCount, true
	}
	if s.BillToAddress != nil {
		p.BillToAddress = customerAddressToProto(s.BillToAddress)
	}
	if s.ShipToAddress != nil {
		p.ShipToAddress = customerAddressToProto(s.ShipToAddress)
	}
	return p
}

func (h *gRPCHandler) BatchGetSuppliersByIDs(ctx context.Context, req *pb.BatchGetSuppliersByIDsRequest) (*pb.BatchGetSuppliersByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	suppliers, apiErr := h.supplierSvc.BatchGetSuppliersByIDs(ctx, req.Ids, nil)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	result := make([]*pb.SupplierSummaryProto, len(suppliers))
	for i, s := range suppliers {
		result[i] = supplierSummaryToProto(s)
	}
	return &pb.BatchGetSuppliersByIDsResponse{Suppliers: result}, nil
}

func (h *gRPCHandler) ListSuppliers(ctx context.Context, req *pb.ListSuppliersRequest) (*pb.ListSuppliersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListSuppliersParams{
		Limit:    req.Limit,
		Cursor:   req.Cursor,
		Query:    req.Query,
		ItemIDs:  req.ItemIds,
		IsActive: req.IsActive,
		Includes: req.Includes,
	}
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.Created.After = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.Created.Before = &t
	}

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
		Name:                  req.Name,
		Number:                req.Number,
		Instructions:          req.Instructions,
		Email:                 req.Email,
		Phone:                 req.Phone,
		URL:                   req.Url,
		CarrierBillingAccount: req.CarrierBillingAccount,
		DefaultCarrierID:      req.DefaultCarrierId,
		DefaultServiceLevelID: req.DefaultServiceLevelId,
		PaymentTermID:         req.PaymentTermId,
		ShippingTermID:        req.ShippingTermId,
		Includes:              req.Includes,
	}
	if req.Status != nil {
		params.ArchiveStatus = constants.ArchiveStatus(*req.Status)
	}
	if req.CarrierBillingType != nil {
		t := constants.CarrierBillingType(*req.CarrierBillingType)
		params.CarrierBillingType = &t
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

// addressClearable reads a default address write: the clearable patch, or the older set-only field.
func addressClearable(patch *pb.StringPatch, set *string) field.Clearable[string] {
	if patch != nil {
		return fieldpb.StringClearableFromProto(patch)
	}
	if set != nil {
		return field.Set(*set)
	}
	return field.Unset[string]()
}

func (h *gRPCHandler) UpdateSupplier(ctx context.Context, req *pb.UpdateSupplierRequest) (*pb.UpdateSupplierResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateSupplierParams{
		SupplierID:            req.Id,
		Name:                  req.Name,
		Number:                req.Number,
		Instructions:          fieldpb.StringClearableFromProto(req.Instructions),
		Email:                 fieldpb.StringClearableFromProto(req.Email),
		Phone:                 fieldpb.StringClearableFromProto(req.Phone),
		URL:                   fieldpb.StringClearableFromProto(req.Url),
		CarrierBillingAccount: fieldpb.StringClearableFromProto(req.CarrierBillingAccount),
		DefaultCarrierID:      fieldpb.StringClearableFromProto(req.DefaultCarrierId),
		DefaultServiceLevelID: fieldpb.StringClearableFromProto(req.DefaultServiceLevelId),
		PaymentTermID:         fieldpb.StringClearableFromProto(req.PaymentTermId),
		ShippingTermID:        fieldpb.StringClearableFromProto(req.ShippingTermId),
		BillToAddressID:       addressClearable(req.DefaultBillToAddressId, req.BillToAddressId),
		ShipToAddressID:       addressClearable(req.DefaultShipToAddressId, req.ShipToAddressId),
		Includes:              req.Includes,
	}
	if req.Status != nil {
		s := constants.ArchiveStatus(*req.Status)
		params.ArchiveStatus = &s
	}
	if req.CarrierBillingType != nil {
		t := constants.CarrierBillingType(*req.CarrierBillingType)
		params.CarrierBillingType = &t
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

func supplierItemToProto(s *domain.SupplierItem) *pb.SupplierItemInfo {
	return &pb.SupplierItemInfo{
		Id:                  s.ID,
		SupplierId:          s.SupplierAccountID,
		ItemId:              s.ItemID,
		SupplierSku:         s.SupplierSKU,
		SupplierDescription: s.SupplierDescription,
		Status:              string(s.Status),
		CreatedAt:           timestamppb.New(s.CreatedAt),
		UpdatedAt:           timestamppb.New(s.UpdatedAt),
	}
}

func (h *gRPCHandler) ListSupplierItems(ctx context.Context, req *pb.ListSupplierItemsRequest) (*pb.ListSupplierItemsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	params := domain.ListSupplierItemsParams{
		SupplierAccountID: req.SupplierId,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Query:             req.Query,
		IsActive:          req.IsActive,
		ItemIDs:           req.ItemIds,
	}
	if req.CreatedAfter != nil {
		t := req.CreatedAfter.AsTime()
		params.Created.After = &t
	}
	if req.CreatedBefore != nil {
		t := req.CreatedBefore.AsTime()
		params.Created.Before = &t
	}
	result, apiErr := h.supplierMaterialSvc.ListSupplierItems(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	items := make([]*pb.SupplierItemInfo, len(result.SupplierItems))
	for i, s := range result.SupplierItems {
		items[i] = supplierItemToProto(s)
	}
	return &pb.ListSupplierItemsResponse{SupplierItems: items, PageInfo: pageInfoToProto(result.PageInfo)}, nil
}

func (h *gRPCHandler) GetSupplierItem(ctx context.Context, req *pb.GetSupplierItemRequest) (*pb.SupplierItemResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	item, apiErr := h.supplierMaterialSvc.GetSupplierItem(ctx, req.SupplierId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.SupplierItemResponse{SupplierItem: supplierItemToProto(item)}, nil
}

func (h *gRPCHandler) CreateSupplierItem(ctx context.Context, req *pb.CreateSupplierItemRequest) (*pb.SupplierItemResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateSupplierItemParams{
		SupplierAccountID:   req.SupplierId,
		ItemID:              req.ItemId,
		SupplierSKU:         req.SupplierSku,
		SupplierDescription: req.SupplierDescription,
	}
	if req.Status != nil {
		params.ArchiveStatus = constants.ArchiveStatus(*req.Status)
	}
	item, apiErr := h.supplierMaterialSvc.CreateSupplierItem(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.SupplierItemResponse{SupplierItem: supplierItemToProto(item)}, nil
}

func (h *gRPCHandler) UpdateSupplierItem(ctx context.Context, req *pb.UpdateSupplierItemRequest) (*pb.SupplierItemResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateSupplierItemParams{
		SupplierAccountID:   req.SupplierId,
		SupplierItemID:      req.Id,
		SupplierSKU:         req.SupplierSku,
		SupplierDescription: fieldpb.StringClearableFromProto(req.SupplierDescription),
	}
	if req.Status != nil {
		s := constants.ArchiveStatus(*req.Status)
		params.ArchiveStatus = &s
	}
	item, apiErr := h.supplierMaterialSvc.UpdateSupplierItem(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.SupplierItemResponse{SupplierItem: supplierItemToProto(item)}, nil
}

func (h *gRPCHandler) DeleteSupplierItem(ctx context.Context, req *pb.DeleteSupplierItemRequest) (*pb.SupplierItemResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	item, apiErr := h.supplierMaterialSvc.DeleteSupplierItem(ctx, req.SupplierId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.SupplierItemResponse{SupplierItem: supplierItemToProto(item)}, nil
}
