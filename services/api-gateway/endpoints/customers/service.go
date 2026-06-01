package customerep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
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

var customerIncludes = []string{
	"bill_to_address", "ship_to_address", "type", "parent_account",
	"freight_preferences", "freight_preferences.carrier", "freight_preferences.service_level",
	"defaults", "defaults.payment_term", "defaults.shipping_term", "defaults.priority", "defaults.sales_rep",
	"contact_info", "notification_preferences", "price_groups", "child_accounts", "credit_limit",
}

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
		Includes:              customerIncludes,
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

	if resp == nil {
		return apiresource.NewList[apiresource.Customer](nil, apiresource.PageInfo{}), nil
	}
	items := make([]apiresource.Customer, len(resp.Customers))
	for i, c := range resp.Customers {
		items[i] = customerFromProto(c)
		stashCustomerMeta(ctx, &items[i], c)
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *customerSvcImpl) GetCustomer(ctx context.Context, req *RetrieveCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
	pbReq := &pb.GetCustomerRequest{
		Id:       req.CustomerID,
		Includes: customerIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCustomerResponse, error) {
			return m.coreClient.GetCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := customerFromProto(resp.Customer)
	stashCustomerMeta(ctx, &result, resp.Customer)
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
		Includes:              customerIncludes,
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

	result := customerFromProto(resp.Customer)
	stashCustomerMeta(ctx, &result, resp.Customer)
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
		Includes:          customerIncludes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.merge", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MergeCustomersResponse, error) {
			return m.coreClient.MergeCustomers(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := customerFromProto(resp.Customer)
	stashCustomerMeta(ctx, &result, resp.Customer)
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
		Includes:                 customerIncludes,
	}

	pbReq.CreditLimit = apirequest.QuantityFieldPtrToProto(req.CreditLimit)

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateCustomerResponse, error) {
			return m.coreClient.UpdateCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := customerFromProto(resp.Customer)
	stashCustomerMeta(ctx, &result, resp.Customer)
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

// --- inline presenter functions ---

func customerFromProto(c *pb.CustomerProto) apiresource.Customer {
	if c == nil {
		return apiresource.Customer{}
	}

	return apiresource.Customer{
		ID:               c.Id,
		Object:           constants.ObjectTypeCustomer,
		Name:             c.Name,
		Number:           c.Number,
		Status:           constants.AccountStatusCode(c.Status),
		EDIStatus:        ediStatusFromBool(c.IsEdiEnabled),
		RelationshipType: customerRelationshipType(c.IsParentAccount, c.ParentAccount != nil),
		CommissionPolicy: constants.CommissionPolicy(c.CommissionPolicy),
		Note:             c.Note,
		CreatedAt:        grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(c.UpdatedAt),
	}
}

func stashCustomerMeta(ctx context.Context, cust *apiresource.Customer, c *pb.CustomerProto) {
	if c == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)

	meta.Set(constants.ObjectTypeCustomer, cust.ID, "contact_info", &apiresource.CustomerContactInfo{
		Object: constants.ObjectTypeCustomerContactInfo,
		Email:  c.Email,
		Phone:  c.Phone,
		URL:    c.Url,
	})

	fp := buildFreightPreferences(c)
	meta.Set(constants.ObjectTypeCustomer, cust.ID, "freight_preferences", fp)
	if c.DefaultCarrier != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "fp_carrier", fp.Carrier)
		fp.Carrier = nil
	}
	if c.DefaultServiceLevel != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "fp_service_level", fp.ServiceLevel)
		fp.ServiceLevel = nil
	}

	defaults := buildDefaults(c)
	meta.Set(constants.ObjectTypeCustomer, cust.ID, "defaults", defaults)
	if defaults.PaymentTerm != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "defaults_payment_term", defaults.PaymentTerm)
		defaults.PaymentTerm = nil
	}
	if defaults.ShippingTerm != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "defaults_shipping_term", defaults.ShippingTerm)
		defaults.ShippingTerm = nil
	}
	if defaults.Priority != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "defaults_priority", defaults.Priority)
		defaults.Priority = nil
	}
	if defaults.SalesRep != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "defaults_sales_rep", defaults.SalesRep)
		defaults.SalesRep = nil
	}

	meta.Set(constants.ObjectTypeCustomer, cust.ID, "notification_preferences", &apiresource.CustomerNotificationPreferences{
		Object:               constants.ObjectTypeCustomerNotificationPreferences,
		AcceptsInvoiceEmails: c.AcceptsInvoiceEmails,
	})

	if c.CreditLimit != nil {
		unitType := c.CreditLimit.UnitType
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "credit_limit", &apiresource.Quantity{
			ID:           c.CreditLimit.Id,
			Object:       constants.ObjectTypeQuantity,
			Value:        apiresource.NormalizeQuantityValue(c.CreditLimit.Value, unitType),
			DisplayValue: apiresource.FormatDisplayValue(c.CreditLimit.Value, c.CreditLimit.UnitAbbreviation, unitType),
			Unit: apiresource.ExpandableUnitStub(
				c.CreditLimit.UnitId,
				c.CreditLimit.UnitName,
				c.CreditLimit.UnitAbbreviation,
				c.CreditLimit.UnitType,
				grpcutil.TimestampToTime(c.CreatedAt),
			),
		})
	}

	if c.BillToAddress != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "bill_to_address", buildCustomerAddress(c.BillToAddress))
	}

	if c.ShipToAddress != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "ship_to_address", buildCustomerAddress(c.ShipToAddress))
	}

	if c.TypeGroup != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "type", buildAccountGroupFromProto(c.TypeGroup))
	}

	priceGroups := make([]apiresource.AccountGroup, len(c.PriceGroups))
	for i, pg := range c.PriceGroups {
		priceGroups[i] = buildAccountGroupValueFromProto(pg)
	}
	meta.Set(constants.ObjectTypeCustomer, cust.ID, "price_groups",
		apiresource.NewList(priceGroups, apiresource.PageInfo{}))

	if c.ParentAccount != nil {
		pa := &apiresource.Customer{
			ID:               c.ParentAccount.Id,
			Object:           constants.ObjectTypeCustomer,
			Name:             c.ParentAccount.Name,
			Number:           c.ParentAccount.Number,
			Status:           constants.AccountStatusCodeNormal,
			EDIStatus:        constants.EDIStatusDisabled,
			RelationshipType: constants.CustomerRelationshipTypeParent,
			CommissionPolicy: constants.CommissionPolicyApplied,
		}
		if c.ParentAccount.CreatedAt != nil {
			pa.CreatedAt = c.ParentAccount.CreatedAt.AsTime()
		}
		if c.ParentAccount.UpdatedAt != nil {
			pa.UpdatedAt = c.ParentAccount.UpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "parent_account", pa)
	}

	if len(c.ChildAccounts) > 0 {
		children := make([]apiresource.Customer, len(c.ChildAccounts))
		for i, child := range c.ChildAccounts {
			ca := apiresource.Customer{
				ID:               child.Id,
				Object:           constants.ObjectTypeCustomer,
				Name:             child.Name,
				Number:           child.Number,
				Status:           constants.AccountStatusCodeNormal,
				EDIStatus:        constants.EDIStatusDisabled,
				RelationshipType: constants.CustomerRelationshipTypeChild,
				CommissionPolicy: constants.CommissionPolicyApplied,
			}
			if child.CreatedAt != nil {
				ca.CreatedAt = child.CreatedAt.AsTime()
			}
			if child.UpdatedAt != nil {
				ca.UpdatedAt = child.UpdatedAt.AsTime()
			}
			children[i] = ca
		}
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "child_accounts",
			apiresource.NewList(children, apiresource.PageInfo{}))
	}
}

// --- helpers ---

func buildFreightPreferences(c *pb.CustomerProto) *apiresource.CustomerFreightPreferences {
	fp := &apiresource.CustomerFreightPreferences{
		Object: constants.ObjectTypeCustomerFreightPreferences,
		Status: constants.FreightPolicy(c.FreightPolicy),
	}

	if c.DefaultCarrier != nil {
		carrierVisibility := constants.CustomerPortalVisibilityHidden
		if c.DefaultCarrier.IsPortalEnabled {
			carrierVisibility = constants.CustomerPortalVisibilityVisible
		}
		carrier := &apiresource.Carrier{
			ID:                       c.DefaultCarrier.Id,
			Object:                   constants.ObjectTypeCarrier,
			Name:                     c.DefaultCarrier.Name,
			CustomerPortalVisibility: carrierVisibility,
		}
		if c.DefaultCarrier.CreatedAt != nil {
			carrier.CreatedAt = c.DefaultCarrier.CreatedAt.AsTime()
		}
		if c.DefaultCarrier.UpdatedAt != nil {
			carrier.UpdatedAt = c.DefaultCarrier.UpdatedAt.AsTime()
		}
		fp.Carrier = carrier
	}

	if c.DefaultServiceLevel != nil {
		sl := &apiresource.ServiceLevel{
			ID:     c.DefaultServiceLevel.Id,
			Object: constants.ObjectTypeServiceLevel,
			Name:   c.DefaultServiceLevel.Name,
		}
		if c.DefaultServiceLevel.ServiceLevelToken != nil {
			sl.ServiceLevelToken = constants.ServiceLevelCode(*c.DefaultServiceLevel.ServiceLevelToken)
		}
		if c.DefaultServiceLevel.IsPortalEnabled != nil && *c.DefaultServiceLevel.IsPortalEnabled {
			sl.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			sl.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if c.DefaultServiceLevel.CreatedAt != nil {
			sl.CreatedAt = c.DefaultServiceLevel.CreatedAt.AsTime()
		}
		if c.DefaultServiceLevel.UpdatedAt != nil {
			sl.UpdatedAt = c.DefaultServiceLevel.UpdatedAt.AsTime()
		}
		fp.ServiceLevel = sl
	}

	if c.CarrierBillingType != nil {
		bt := constants.CarrierBillingType(*c.CarrierBillingType)
		fp.BillingType = &bt
	}

	fp.BillingAccount = c.CarrierBillingAccount
	return fp
}

func buildDefaults(c *pb.CustomerProto) *apiresource.CustomerDefaults {
	d := &apiresource.CustomerDefaults{Object: constants.ObjectTypeCustomerDefaults}

	if c.DefaultPaymentTerm != nil {
		ptStatus := constants.PaymentTermStatusInactive
		if c.DefaultPaymentTerm.IsActive {
			ptStatus = constants.PaymentTermStatusActive
		}
		pt := &apiresource.PaymentTerm{
			ID:     c.DefaultPaymentTerm.Id,
			Object: constants.ObjectTypePaymentTerm,
			Name:   c.DefaultPaymentTerm.Name,
			Status: ptStatus,
		}
		if c.DefaultPaymentTerm.CreatedAt != nil {
			pt.CreatedAt = c.DefaultPaymentTerm.CreatedAt.AsTime()
		}
		if c.DefaultPaymentTerm.UpdatedAt != nil {
			pt.UpdatedAt = c.DefaultPaymentTerm.UpdatedAt.AsTime()
		}
		d.PaymentTerm = pt
	}

	if c.DefaultShippingTerm != nil {
		st := &apiresource.ShippingTerm{
			ID:     c.DefaultShippingTerm.Id,
			Object: constants.ObjectTypeShippingTerm,
			Name:   c.DefaultShippingTerm.Name,
			Type:   constants.ShippingTermType(c.DefaultShippingTerm.Type),
		}
		if c.DefaultShippingTerm.CreatedAt != nil {
			st.CreatedAt = c.DefaultShippingTerm.CreatedAt.AsTime()
		}
		if c.DefaultShippingTerm.UpdatedAt != nil {
			st.UpdatedAt = c.DefaultShippingTerm.UpdatedAt.AsTime()
		}
		d.ShippingTerm = st
	}

	if c.DefaultPriority != nil {
		d.Priority = apiresource.ExpandablePriorityStub(
			c.DefaultPriority.Id,
			constants.PriorityCode(c.DefaultPriority.Code),
			c.DefaultPriority.Name,
			grpcutil.TimestampToTime(c.CreatedAt),
		)
	}

	if c.DefaultSalesRep != nil {
		sr := &apiresource.AccountUser{
			ID:     c.DefaultSalesRep.Id,
			Object: constants.ObjectTypeAccountUser,
			Name:   c.DefaultSalesRep.Name,
		}
		if c.DefaultSalesRep.Status != nil {
			sr.Status = constants.AccountUserStatus(*c.DefaultSalesRep.Status)
		}
		if c.DefaultSalesRep.CreatedAt != nil {
			sr.CreatedAt = c.DefaultSalesRep.CreatedAt.AsTime()
		}
		if c.DefaultSalesRep.UpdatedAt != nil {
			sr.UpdatedAt = c.DefaultSalesRep.UpdatedAt.AsTime()
		}
		d.SalesRep = sr
	}

	return d
}

func buildCustomerAddress(a *pb.CustomerAddressProto) *apiresource.Address {
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

func buildAccountGroupFromProto(g *pb.CustomerAccountGroupProto) *apiresource.AccountGroup {
	ag := &apiresource.AccountGroup{
		ID:               g.Id,
		Object:           constants.ObjectTypeAccountGroup,
		Name:             g.Name,
		CommissionPolicy: constants.CommissionPolicy(g.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(g.FreightPolicy),
		Type:             constants.AccountGroupType(g.Type),
	}
	if g.CreatedAt != nil {
		ag.CreatedAt = g.CreatedAt.AsTime()
	}
	if g.UpdatedAt != nil {
		ag.UpdatedAt = g.UpdatedAt.AsTime()
	}
	return ag
}

func buildAccountGroupValueFromProto(g *pb.CustomerAccountGroupProto) apiresource.AccountGroup {
	ag := apiresource.AccountGroup{
		ID:               g.Id,
		Object:           constants.ObjectTypeAccountGroup,
		Name:             g.Name,
		CommissionPolicy: constants.CommissionPolicy(g.CommissionPolicy),
		FreightPolicy:    constants.FreightPolicy(g.FreightPolicy),
		Type:             constants.AccountGroupType(g.Type),
	}
	if g.CreatedAt != nil {
		ag.CreatedAt = g.CreatedAt.AsTime()
	}
	if g.UpdatedAt != nil {
		ag.UpdatedAt = g.UpdatedAt.AsTime()
	}
	return ag
}

func ediStatusFromBool(enabled bool) constants.EDIStatus {
	if enabled {
		return constants.EDIStatusEnabled
	}
	return constants.EDIStatusDisabled
}

func customerRelationshipType(isParent bool, hasParent bool) constants.CustomerRelationshipType {
	if isParent {
		return constants.CustomerRelationshipTypeParent
	}
	if hasParent {
		return constants.CustomerRelationshipTypeChild
	}
	return constants.CustomerRelationshipTypeStandalone
}

func addressTypeFromDropShip(isDropShip bool) constants.AddressType {
	if isDropShip {
		return constants.AddressTypeDropShip
	}
	return constants.AddressTypeStandard
}
