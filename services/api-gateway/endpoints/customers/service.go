package customerep

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
	"github.com/augno/api/shared/patch"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func optAccountStatusCodeToStringPtr(p *constants.AccountStatusCode) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func optPriorityCodeToStringPtr(p *constants.PriorityCode) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func optCarrierBillingTypeToStringPtr(p *constants.CarrierBillingType) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func accountStatusCodesToStrings(codes []constants.AccountStatusCode) []string {
	out := make([]string, len(codes))
	for i, c := range codes {
		out[i] = string(c)
	}
	return out
}

func commissionPoliciesToStrings(codes []constants.CommissionPolicy) []string {
	out := make([]string, len(codes))
	for i, c := range codes {
		out[i] = string(c)
	}
	return out
}

func freightPoliciesToStrings(codes []constants.FreightPolicy) []string {
	out := make([]string, len(codes))
	for i, c := range codes {
		out[i] = string(c)
	}
	return out
}

func optCommissionPolicyToStringPtr(p *constants.CommissionPolicy) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func optFreightPolicyToStringPtr(p *constants.FreightPolicy) *string {
	if p == nil {
		return nil
	}
	s := string(*p)
	return &s
}

func ediStatusToBoolPtr(s *constants.EDIStatus) *bool {
	if s == nil {
		return nil
	}
	v := *s == constants.EDIStatusEnabled
	return &v
}

func addressTypeToDropShip(t *constants.AddressType) bool {
	return t != nil && *t == constants.AddressTypeDropShip
}

func parentAccountStatusToBoolPtr(status *constants.CustomerParentAccountStatus) *bool {
	if status == nil {
		return nil
	}
	v := *status == constants.CustomerParentAccountStatusParent
	return &v
}

func derefStringSlice(p *[]string) []string {
	if p == nil {
		return nil
	}
	return *p
}

type CustomerSvc interface {
	ListCustomers(ctx context.Context, req *ListCustomersRequest) (*apiresource.List[apiresource.Customer], *apierror.APIError)
	GetCustomer(ctx context.Context, req *RetrieveCustomerRequest) (*apiresource.Customer, *apierror.APIError)
	CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*apiresource.Customer, *apierror.APIError)
	DeleteCustomer(ctx context.Context, req *DeleteCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkDeleteCustomers(ctx context.Context, req *BulkDeleteCustomersRequest) (*apiresource.EmptyResource, *apierror.APIError)
	GetFrequentlyOrderedProducts(ctx context.Context, req *GetFrequentlyOrderedProductsRequest) (*apiresource.List[apiresource.FrequentlyOrderedProduct], *apierror.APIError)
	MergeCustomers(ctx context.Context, req *MergeCustomersRequest) (*apiresource.Customer, *apierror.APIError)
	UpdateCustomer(ctx context.Context, req *UpdateCustomerRequest) (*apiresource.Customer, *apierror.APIError)
}

type CustomerSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type customerSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var customerSvcTracer = tracing.GetTracer("api-gateway.endpoints.customers.service")

func (c *CustomerSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("customer endpoint service: core client is required")
	}
	return nil
}

func NewCustomerSvc(config *CustomerSvcConfig) CustomerSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &customerSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *customerSvcImpl) ListCustomers(ctx context.Context, req *ListCustomersRequest) (*apiresource.List[apiresource.Customer], *apierror.APIError) {
	pbReq := &pb.ListCustomersRequest{
		Cursor:                req.Cursor,
		Limit:                 req.Limit,
		Query:                 req.Query,
		CustomerGroupIds:      req.CustomerGroupIDs,
		PricingGroupIds:       req.PricingGroupIDs,
		SalesRepIds:           req.SalesRepIDs,
		StatusCodes:           accountStatusCodesToStrings(req.StatusCodes),
		ShippingTermIds:       req.ShippingTermIDs,
		PaymentTermIds:        req.PaymentTermIDs,
		CommissionStatusCodes: commissionPoliciesToStrings(req.CommissionPolicyCodes),
		FreightStatusCodes:    freightPoliciesToStrings(req.FreightPolicyCodes),
		CarrierIds:            req.CarrierIDs,
		ServiceLevelIds:       req.ServiceLevelIDs,
		IsParentAccount:       parentAccountStatusToBoolPtr(req.ParentAccountStatus),
		City:                  req.City,
		State:                 req.State,
		PostalCode:            req.PostalCode,
		Includes:              appctx.GetRequestedIncludeKeys(ctx),
	}

	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCustomersResponse, error) {
			return m.coreClient.ListCustomers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return CustomerListPresenter(ctx, resp), nil
}

func (m *customerSvcImpl) GetCustomer(ctx context.Context, req *RetrieveCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
	pbReq := &pb.GetCustomerRequest{
		Id:       req.CustomerID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCustomerResponse, error) {
			return m.coreClient.GetCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CustomerPresenter(resp.Customer)
	return &result, nil
}

func (m *customerSvcImpl) CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
	statusCode := constants.AccountStatusCodeNormal
	if req.StatusCode != nil {
		statusCode = *req.StatusCode
	}
	commissionPolicy := constants.CommissionPolicyExempt
	if req.CommissionPolicy != nil {
		commissionPolicy = *req.CommissionPolicy
	}
	freightPolicy := constants.FreightPolicyBilled
	if req.FreightPolicy != nil {
		freightPolicy = *req.FreightPolicy
	}
	priorityCode := constants.PriorityCodeNormal
	if req.DefaultPriorityCode != nil {
		priorityCode = *req.DefaultPriorityCode
	}

	statusCodeStr := string(statusCode)
	commissionPolicyStr := string(commissionPolicy)
	freightPolicyStr := string(freightPolicy)
	priorityCodeStr := string(priorityCode)

	pbReq := &pb.CreateCustomerRequest{
		Name:                  req.Name,
		Number:                req.Number,
		Note:                  req.Note,
		Email:                 req.Email,
		Phone:                 req.Phone,
		Url:                   req.URL,
		StatusCode:            &statusCodeStr,
		IsEdiEnabled:          ediStatusToBoolPtr(req.EDIStatus),
		CommissionPolicy:      &commissionPolicyStr,
		FreightPolicy:         &freightPolicyStr,
		DefaultCarrierId:      &req.DefaultCarrierID,
		DefaultServiceLevelId: req.DefaultServiceLevelID,
		DefaultPaymentTermId:  &req.DefaultPaymentTermID,
		DefaultShippingTermId: &req.DefaultShippingTermID,
		DefaultPriorityCode:   &priorityCodeStr,
		DefaultSalesRepId:     req.DefaultSalesRepID,
		CustomerPriceGroupIds: req.CustomerPriceGroupIDs,
		CustomerTypeGroupId:   &req.CustomerTypeGroupID,
		CarrierBillingType:    optCarrierBillingTypeToStringPtr(req.CarrierBillingType),
		CarrierBillingAccount: req.CarrierBillingAccount,
		Includes:              appctx.GetRequestedIncludeKeys(ctx),
	}

	if req.CreditLimit != nil {
		pbReq.CreditLimitValue = &req.CreditLimit.Value
		pbReq.CreditLimitUnitId = &req.CreditLimit.UnitID
	}

	pbReq.BillToAddress = addressInputToCustomerProto(&req.BillToAddress)
	pbReq.ShipToAddress = addressInputToCustomerProto(&req.ShipToAddress)

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCustomerResponse, error) {
			return m.coreClient.CreateCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CustomerPresenter(resp.Customer)
	return &result, nil
}

func (m *customerSvcImpl) DeleteCustomer(ctx context.Context, req *DeleteCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteCustomerRequest{
		Id: req.CustomerID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *customerSvcImpl) BulkDeleteCustomers(ctx context.Context, req *BulkDeleteCustomersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.BulkDeleteCustomersRequest{
		CustomerIds: req.CustomerIDs,
	}

	_, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.bulk_delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.BulkDeleteCustomers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *customerSvcImpl) GetFrequentlyOrderedProducts(ctx context.Context, req *GetFrequentlyOrderedProductsRequest) (*apiresource.List[apiresource.FrequentlyOrderedProduct], *apierror.APIError) {
	pbReq := &pb.GetFrequentlyOrderedProductsRequest{
		CustomerId: req.CustomerID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.frequently_ordered_products", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetFrequentlyOrderedProductsResponse, error) {
			return m.coreClient.GetFrequentlyOrderedProducts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	products := make([]apiresource.FrequentlyOrderedProduct, len(resp.Products))
	for i, p := range resp.Products {
		fop := apiresource.FrequentlyOrderedProduct{
			Object: constants.ObjectTypeFrequentlyOrderedProduct,
			Item: &apiresource.Item{
				ID:     p.ItemId,
				Object: constants.ObjectTypeItem,
				SKU:    p.ProductName,
			},
			OrderCount: p.OrderCount,
		}
		if p.UnitId != nil {
			fop.Unit = &apiresource.Unit{
				ID:     *p.UnitId,
				Object: constants.ObjectTypeUnit,
			}
			if p.UnitAbbreviation != nil {
				fop.Unit.Abbreviation = *p.UnitAbbreviation
			}
		}
		products[i] = fop
	}

	return apiresource.NewList(products, apiresource.PageInfo{}), nil
}

func (m *customerSvcImpl) MergeCustomers(ctx context.Context, req *MergeCustomersRequest) (*apiresource.Customer, *apierror.APIError) {
	pbReq := &pb.MergeCustomersRequest{
		TargetCustomerId:  req.CustomerID,
		SourceCustomerIds: req.SourceCustomerIDs,
		Includes:          appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.merge", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MergeCustomersResponse, error) {
			return m.coreClient.MergeCustomers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CustomerPresenter(resp.Customer)
	return &result, nil
}

func (m *customerSvcImpl) UpdateCustomer(ctx context.Context, req *UpdateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
	pbReq := &pb.UpdateCustomerRequest{
		Id:                       req.CustomerID,
		Name:                     req.Name,
		Number:                   req.Number,
		Note:                     patch.StringFieldPtrToProto(req.Note),
		Email:                    patch.StringFieldPtrToProto(req.Email),
		Phone:                    patch.StringFieldPtrToProto(req.Phone),
		Url:                      patch.StringFieldPtrToProto(req.URL),
		StatusCode:               optAccountStatusCodeToStringPtr(req.StatusCode),
		IsEdiEnabled:             ediStatusToBoolPtr(req.EDIStatus),
		CommissionPolicy:         optCommissionPolicyToStringPtr(req.CommissionPolicy),
		FreightPolicy:            optFreightPolicyToStringPtr(req.FreightPolicy),
		DefaultCarrierId:         req.DefaultCarrierID,
		DefaultServiceLevelId:    patch.StringFieldPtrToProto(req.DefaultServiceLevelID),
		DefaultPaymentTermId:     req.DefaultPaymentTermID,
		DefaultShippingTermId:    req.DefaultShippingTermID,
		DefaultPriorityCode:      optPriorityCodeToStringPtr(req.DefaultPriorityCode),
		DefaultSalesRepId:        patch.StringFieldPtrToProto(req.DefaultSalesRepID),
		BillToAddressId:          patch.StringFieldPtrToProto(req.BillToAddressID),
		ShipToAddressId:          patch.StringFieldPtrToProto(req.ShipToAddressID),
		CustomerPriceGroupIds:    derefStringSlice(req.CustomerPriceGroupIDs),
		CustomerTypeGroupId:      req.CustomerTypeGroupID,
		CarrierBillingType:       optCarrierBillingTypeToStringPtr(req.CarrierBillingType),
		CarrierBillingAccount:    patch.StringFieldPtrToProto(req.CarrierBillingAccount),
		HasCustomerPriceGroupIds: req.CustomerPriceGroupIDs != nil,
		Includes:                 appctx.GetRequestedIncludeKeys(ctx),
	}

	pbReq.CreditLimit = apirequest.QuantityFieldPtrToProto(req.CreditLimit)

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateCustomerResponse, error) {
			return m.coreClient.UpdateCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CustomerPresenter(resp.Customer)
	return &result, nil
}

func addressInputToCustomerProto(a *apirequest.AddressInput) *pb.CreateCustomerAddressInput {
	if a == nil {
		return nil
	}
	return &pb.CreateCustomerAddressInput{
		Name:         a.Name,
		Phone:        a.Phone.Ptr(),
		Email:        a.Email.Ptr(),
		IsDropShip:   addressTypeToDropShip(a.Type),
		StreetLine_1: a.StreetLine1.Ptr(),
		StreetLine_2: a.StreetLine2.Ptr(),
		Locality:     a.Locality.Ptr(),
		State:        a.State.Ptr(),
		PostalCode:   a.PostalCode.Ptr(),
		Country:      a.Country,
	}
}
