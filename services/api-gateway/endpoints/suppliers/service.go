package supplierep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
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
	if apiErr := req.CreatedRange.Validate(); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.ListSuppliersRequest{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		ItemIds:  req.ItemIDs,
		IsActive: constants.IsActiveFilter(req.Statuses),
		Includes: resourceloaders.SupplierCoreIncludes(ctx),
	}
	if req.CreatedAfter != nil {
		pbReq.StartDate = timestamppb.New(*req.CreatedAfter)
	}
	if req.CreatedBefore != nil {
		pbReq.EndDate = timestamppb.New(*req.CreatedBefore)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSuppliersResponse, error) {
			return s.coreClient.ListSuppliers(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if resp == nil {
		return apiresource.NewList[apiresource.Supplier](nil, apiresource.PageInfo{}), nil
	}

	items := make([]apiresource.Supplier, len(resp.Suppliers))
	for i, sp := range resp.Suppliers {
		items[i] = *resourceloaders.SupplierFromSummaryProto(ctx, sp)
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (s *supplierSvcImpl) GetSupplier(ctx context.Context, req *RetrieveSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
	pbReq := &pb.GetSupplierRequest{
		Id:       req.SupplierID,
		Includes: resourceloaders.SupplierCoreIncludes(ctx),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSupplierResponse, error) {
			return s.coreClient.GetSupplier(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.SupplierFromProto(ctx, resp.Supplier), nil
}

func (s *supplierSvcImpl) CreateSupplier(ctx context.Context, req *CreateSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
	pbReq := &pb.CreateSupplierRequest{
		Name:         req.Name,
		Instructions: req.Instructions.Ptr(),
		Includes:     resourceloaders.SupplierCoreIncludes(ctx),
	}
	pbReq.Number, _ = req.Number.Value()
	if v, ok := req.Status.Value(); ok {
		status := string(v)
		pbReq.Status = &status
	}
	if contact, ok := req.Contact.Value(); ok {
		pbReq.Email, pbReq.Phone, pbReq.Url = contact.Email.ValuePtr(), contact.Phone.ValuePtr(), contact.URL.ValuePtr()
	}
	if billing, ok := req.CarrierBilling.Value(); ok {
		if t, ok := billing.Type.Value(); ok {
			v := string(t)
			pbReq.CarrierBillingType = &v
		}
		pbReq.CarrierBillingAccount = billing.AccountNumber.ValuePtr()
	}
	if defaults, ok := req.Defaults.Value(); ok {
		pbReq.DefaultCarrierId = defaults.CarrierID.Ptr()
		pbReq.DefaultServiceLevelId = defaults.ServiceLevelID.Ptr()
		pbReq.PaymentTermId = defaults.PaymentTermID.Ptr()
		pbReq.ShippingTermId = defaults.ShippingTermID.Ptr()
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
	return resourceloaders.SupplierFromProto(ctx, resp.Supplier), nil
}

func (s *supplierSvcImpl) UpdateSupplier(ctx context.Context, req *UpdateSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
	pbReq := &pb.UpdateSupplierRequest{
		Id:           req.SupplierID,
		Name:         req.Name.Ptr(),
		Number:       req.Number.Ptr(),
		Instructions: fieldpb.StringClearableToProto(req.Instructions),
		Includes:     resourceloaders.SupplierCoreIncludes(ctx),
	}
	if v, ok := req.Status.Value(); ok {
		status := string(v)
		pbReq.Status = &status
	}
	if contact, ok := req.Contact.Value(); ok {
		pbReq.Email = fieldpb.StringClearableToProto(contact.Email)
		pbReq.Phone = fieldpb.StringClearableToProto(contact.Phone)
		pbReq.Url = fieldpb.StringClearableToProto(contact.URL)
	}
	if billing, ok := req.CarrierBilling.Value(); ok {
		if t, ok := billing.Type.Value(); ok {
			v := string(t)
			pbReq.CarrierBillingType = &v
		}
		pbReq.CarrierBillingAccount = fieldpb.StringClearableToProto(billing.AccountNumber)
	}
	if defaults, ok := req.Defaults.Value(); ok {
		pbReq.DefaultCarrierId = fieldpb.StringClearableToProto(defaults.CarrierID)
		pbReq.DefaultServiceLevelId = fieldpb.StringClearableToProto(defaults.ServiceLevelID)
		pbReq.PaymentTermId = fieldpb.StringClearableToProto(defaults.PaymentTermID)
		pbReq.ShippingTermId = fieldpb.StringClearableToProto(defaults.ShippingTermID)
		pbReq.DefaultBillToAddressId = fieldpb.StringClearableToProto(defaults.BillToAddressID)
		pbReq.DefaultShipToAddressId = fieldpb.StringClearableToProto(defaults.ShipToAddressID)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSupplierResponse, error) {
			return s.coreClient.UpdateSupplier(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.SupplierFromProto(ctx, resp.Supplier), nil
}

func (s *supplierSvcImpl) DeleteSupplier(ctx context.Context, req *DeleteSupplierRequest) (*apiresource.Supplier, *apierror.APIError) {
	pbReq := &pb.DeleteSupplierRequest{Id: req.SupplierID}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierSvcTracer, "service.suppliers.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteSupplierResponse, error) {
			return s.coreClient.DeleteSupplier(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.SupplierFromProto(ctx, resp.Supplier), nil
}

func (s *supplierSvcImpl) BulkDeleteSuppliers(ctx context.Context, req *BulkDeleteSuppliersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.BulkDeleteSuppliersRequest{SupplierIds: req.SupplierIDs}
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
		IsDropShip:   a.Type.Ptr() != nil && *a.Type.Ptr() == constants.AddressTypeDropShip,
		StreetLine_1: a.Line1.Ptr(),
		StreetLine_2: a.Line2.Ptr(),
		Locality:     a.City.Ptr(),
		State:        a.State.Ptr(),
		PostalCode:   a.PostalCode.Ptr(),
		Country:      a.Country,
	}
}
