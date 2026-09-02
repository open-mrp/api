package supplierep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SupplierSvc interface {
	ListSuppliers(ctx context.Context, req *ListSuppliersRequest) (*apiresource.List[apiresource.Supplier], *apierror.APIError)
	GetSupplier(ctx context.Context, req *RetrieveSupplierRequest) (*apiresource.Supplier, *apierror.APIError)
	CreateSupplier(ctx context.Context, req *CreateSupplierRequest) (*apiresource.Supplier, *apierror.APIError)
	UpdateSupplier(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.Supplier, *apierror.APIError)
	DeleteSupplier(ctx context.Context, req *DeleteSupplierRequest) (*apiresource.Supplier, *apierror.APIError)
	BulkDeleteSuppliers(ctx context.Context, req *BulkDeleteSuppliersRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type SupplierSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type supplierSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var supplierSvcTracer = tracing.GetTracer("api-gateway.endpoints.suppliers.service")

var supplierIncludes = []string{"bill_to_address", "ship_to_address"}

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

func (s *supplierSvcImpl) ListSuppliers(ctx context.Context, req *ListSuppliersRequest) (*apiresource.List[apiresource.Supplier], *apierror.APIError) {
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

	return supplierListFromProto(ctx, resp), nil
}

func (s *supplierSvcImpl) GetSupplier(ctx context.Context, req *RetrieveSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
	pbReq := &pb.GetSupplierRequest{
		Id:       req.SupplierID,
		Includes: resourcekit.FilterIncludes(ctx, supplierIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSupplierResponse, error) {
			return s.coreClient.GetSupplier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := supplierDetailFromProto(resp.Supplier)
	stashSupplierMeta(ctx, resp.Supplier, &result)
	return &result, nil
}

func (s *supplierSvcImpl) CreateSupplier(ctx context.Context, req *CreateSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
	pbReq := &pb.CreateSupplierRequest{
		Name:     req.Name,
		Number:   req.Number,
		Note:     req.Note.Ptr(),
		Includes: resourcekit.FilterIncludes(ctx, supplierIncludes...),
	}

	if v, ok := req.BillToAddress.Value(); ok {
		pbReq.BillToAddress = createAddressRequestToProto(&v)
	}
	if v, ok := req.ShipToAddress.Value(); ok {
		pbReq.ShipToAddress = createAddressRequestToProto(&v)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSupplierResponse, error) {
			return s.coreClient.CreateSupplier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := supplierDetailFromProto(resp.Supplier)
	stashSupplierMeta(ctx, resp.Supplier, &result)
	return &result, nil
}

func (s *supplierSvcImpl) UpdateSupplier(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
	pbReq := &pb.UpdateSupplierRequest{
		Id:              req.SupplierID,
		Name:            req.Name.Ptr(),
		Number:          req.Number.Ptr(),
		Note:            req.Note.Ptr(),
		UpdateNote:      req.UpdateNote,
		BillToAddressId: req.BillToAddressID.Ptr(),
		ShipToAddressId: req.ShipToAddressID.Ptr(),
		Includes:        resourcekit.FilterIncludes(ctx, supplierIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSupplierResponse, error) {
			return s.coreClient.UpdateSupplier(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := supplierDetailFromProto(resp.Supplier)
	stashSupplierMeta(ctx, resp.Supplier, &result)
	return &result, nil
}

func (s *supplierSvcImpl) DeleteSupplier(ctx context.Context, req *DeleteSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
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

	result := supplierDetailFromProto(resp.Supplier)
	stashSupplierMeta(ctx, resp.Supplier, &result)
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
		Phone:        a.Phone.Ptr(),
		Email:        a.Email.Ptr(),
		IsDropShip:   addressTypeToDropShip(a.Type.Ptr()),
		StreetLine_1: a.StreetLine1.Ptr(),
		StreetLine_2: a.StreetLine2.Ptr(),
		Locality:     a.Locality.Ptr(),
		State:        a.State.Ptr(),
		PostalCode:   a.PostalCode.Ptr(),
		Country:      a.Country,
	}
}

func supplierDetailFromProto(s *pb.SupplierProto) apiresource.Supplier {
	if s == nil {
		return apiresource.Supplier{}
	}

	return apiresource.Supplier{
		ID:            s.Id,
		Object:        constants.ObjectTypeSupplier,
		Name:          s.Name,
		Number:        s.Number,
		Note:          s.Note,
		MaterialCount: s.MaterialCount,
		CreatedAt:     grpcutil.TimestampToTimePtr(s.CreatedAt),
		UpdatedAt:     grpcutil.TimestampToTimePtr(s.UpdatedAt),
	}
}

func stashSupplierMeta(ctx context.Context, s *pb.SupplierProto, d *apiresource.Supplier) {
	if s == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	if s.BillToAddress != nil {
		meta.Set(constants.ObjectTypeSupplier, d.ID, "bill_to_address", addressProtoToResource(s.BillToAddress))
	}
	if s.ShipToAddress != nil {
		meta.Set(constants.ObjectTypeSupplier, d.ID, "ship_to_address", addressProtoToResource(s.ShipToAddress))
	}
}

func supplierSummaryFromProto(s *pb.SupplierSummaryProto) apiresource.Supplier {
	if s == nil {
		return apiresource.Supplier{}
	}

	return apiresource.Supplier{
		ID:            s.Id,
		Object:        constants.ObjectTypeSupplier,
		Name:          s.Name,
		Number:        s.Number,
		MaterialCount: s.MaterialCount,
		CreatedAt:     grpcutil.TimestampToTimePtr(s.CreatedAt),
	}
}

func supplierListFromProto(ctx context.Context, resp *pb.ListSuppliersResponse) *apiresource.List[apiresource.Supplier] {
	if resp == nil {
		return apiresource.NewList[apiresource.Supplier](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.Supplier, len(resp.Suppliers))
	for i, s := range resp.Suppliers {
		items[i] = supplierSummaryFromProto(s)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
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
