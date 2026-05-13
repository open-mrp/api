package supplierep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SupplierSvc interface {
	ListSuppliers(ctx context.Context, req *ListSuppliersRequest) (*apiresource.List[apiresource.SupplierSummary], *apierror.APIError)
	GetSupplier(ctx context.Context, req *RetrieveSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError)
	CreateSupplier(ctx context.Context, req *CreateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError)
	UpdateSupplier(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError)
	DeleteSupplier(ctx context.Context, req *DeleteSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError)
	BulkDeleteSuppliers(ctx context.Context, req *BulkDeleteSuppliersRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type SupplierSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type supplierSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var supplierSvcTracer = tracing.GetTracer("api-gateway.endpoints.suppliers.service")

func (c *SupplierSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("supplier endpoint service: core client is required")
	}
	return nil
}

func NewSupplierSvc(config *SupplierSvcConfig) SupplierSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &supplierSvcImpl{coreClient: config.CoreClient}
}

func (s *supplierSvcImpl) ListSuppliers(ctx context.Context, req *ListSuppliersRequest) (*apiresource.List[apiresource.SupplierSummary], *apierror.APIError) {
	pbReq := &pb.ListSuppliersRequest{
		Cursor:  req.Cursor,
		Limit:   req.Limit,
		Query:   req.Query,
		ItemIds: req.ItemIDs,
	}

	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSuppliersResponse, error) {
			return s.coreClient.ListSuppliers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return SupplierListPresenter(resp), nil
}

func (s *supplierSvcImpl) GetSupplier(ctx context.Context, req *RetrieveSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
	pbReq := &pb.GetSupplierRequest{
		Id:       req.SupplierID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSupplierResponse, error) {
			return s.coreClient.GetSupplier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SupplierPresenter(resp.Supplier)
	return &result, nil
}

func (s *supplierSvcImpl) CreateSupplier(ctx context.Context, req *CreateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
	pbReq := &pb.CreateSupplierRequest{
		Name:     req.Name,
		Number:   req.Number,
		Note:     req.Note,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	if req.BillToAddress != nil {
		pbReq.BillToAddress = createAddressRequestToProto(req.BillToAddress)
	}
	if req.ShipToAddress != nil {
		pbReq.ShipToAddress = createAddressRequestToProto(req.ShipToAddress)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSupplierResponse, error) {
			return s.coreClient.CreateSupplier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SupplierPresenter(resp.Supplier)
	return &result, nil
}

func (s *supplierSvcImpl) UpdateSupplier(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
	pbReq := &pb.UpdateSupplierRequest{
		Id:              req.SupplierID,
		Name:            req.Name,
		Number:          req.Number,
		Note:            req.Note,
		UpdateNote:      req.UpdateNote,
		BillToAddressId: req.BillToAddressID,
		ShipToAddressId: req.ShipToAddressID,
		Includes:        appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSupplierResponse, error) {
			return s.coreClient.UpdateSupplier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SupplierPresenter(resp.Supplier)
	return &result, nil
}

func (s *supplierSvcImpl) DeleteSupplier(ctx context.Context, req *DeleteSupplierRequest) (*apiresource.SupplierDetail, *apierror.APIError) {
	pbReq := &pb.DeleteSupplierRequest{
		Id: req.SupplierID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteSupplierResponse, error) {
			return s.coreClient.DeleteSupplier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := SupplierPresenter(resp.Supplier)
	return &result, nil
}

func (s *supplierSvcImpl) BulkDeleteSuppliers(ctx context.Context, req *BulkDeleteSuppliersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.BulkDeleteSuppliersRequest{
		SupplierIds: req.SupplierIDs,
	}

	_, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.bulk_delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return s.coreClient.BulkDeleteSuppliers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func createAddressRequestToProto(a *apirequest.AddressInput) *pb.CreateSupplierAddressInput {
	if a == nil {
		return nil
	}
	return &pb.CreateSupplierAddressInput{
		Name:         a.Name,
		Phone:        a.Phone,
		Email:        a.Email,
		IsDropShip:   addressTypeToDropShip(a.Type),
		StreetLine_1: a.StreetLine1,
		StreetLine_2: a.StreetLine2,
		Locality:     a.Locality,
		State:        a.State,
		PostalCode:   a.PostalCode,
		Country:      a.Country,
	}
}

// SupplierPresenter converts a SupplierProto to a SupplierDetail API resource.
func SupplierPresenter(s *pb.SupplierProto) apiresource.SupplierDetail {
	if s == nil {
		return apiresource.SupplierDetail{}
	}

	var billToAddress *apiresource.Address
	if s.BillToAddress != nil {
		billToAddress = addressProtoToResource(s.BillToAddress)
	}

	var shipToAddress *apiresource.Address
	if s.ShipToAddress != nil {
		shipToAddress = addressProtoToResource(s.ShipToAddress)
	}

	return apiresource.SupplierDetail{
		ID:            s.Id,
		Object:        constants.ObjectTypeSupplier,
		Name:          s.Name,
		Number:        s.Number,
		Note:          s.Note,
		BillToAddress: billToAddress,
		ShipToAddress: shipToAddress,
		MaterialCount: s.MaterialCount,
		CreatedAt:     grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:     grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

// SupplierSummaryPresenter converts a SupplierSummaryProto to a SupplierSummary API resource.
func SupplierSummaryPresenter(s *pb.SupplierSummaryProto) apiresource.SupplierSummary {
	if s == nil {
		return apiresource.SupplierSummary{}
	}

	return apiresource.SupplierSummary{
		ID:            s.Id,
		Object:        constants.ObjectTypeSupplierSummary,
		Name:          s.Name,
		Number:        s.Number,
		MaterialCount: s.MaterialCount,
		CreatedAt:     grpcutil.TimestampToTime(s.CreatedAt),
	}
}

// SupplierListPresenter converts a ListSuppliersResponse to a list of SupplierSummary API resources.
func SupplierListPresenter(resp *pb.ListSuppliersResponse) *apiresource.List[apiresource.SupplierSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.SupplierSummary](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.SupplierSummary, len(resp.Suppliers))
	for i, s := range resp.Suppliers {
		items[i] = SupplierSummaryPresenter(s)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func addressProtoToResource(a *pb.CustomerAddressProto) *apiresource.Address {
	if a == nil {
		return nil
	}

	var geolocation *apiresource.Geolocation
	if a.Geolocation != nil {
		geolocation = &apiresource.Geolocation{
			ID:          a.Geolocation.Id,
			Object:      constants.ObjectTypeGeolocation,
			StreetLine1: a.Geolocation.StreetLine_1,
			StreetLine2: a.Geolocation.StreetLine_2,
			Locality:    a.Geolocation.Locality,
			State:       a.Geolocation.State,
			PostalCode:  a.Geolocation.PostalCode,
			Country:     a.Geolocation.Country,
		}
	}

	return &apiresource.Address{
		ID:          a.Id,
		Object:      constants.ObjectTypeAddress,
		Name:        a.Name,
		Phone:       a.Phone,
		Email:       a.Email,
		Type:        addressTypeFromDropShip(a.IsDropShip),
		Geolocation: geolocation,
		CreatedAt:   grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func addressTypeToDropShip(t *constants.AddressType) bool {
	return t != nil && *t == constants.AddressTypeDropShip
}

func addressTypeFromDropShip(isDropShip bool) constants.AddressType {
	if isDropShip {
		return constants.AddressTypeDropShip
	}
	return constants.AddressTypeStandard
}
