package customerep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

type CustomerSvc interface {
	ListCustomers(ctx context.Context, req *ListCustomersRequest) (*apiresource.List[apiresource.Customer], *apierror.APIError)
	GetCustomer(ctx context.Context, req *RetrieveCustomerRequest) (*apiresource.Customer, *apierror.APIError)
	CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*apiresource.Customer, *apierror.APIError)
	DeleteCustomer(ctx context.Context, req *DeleteCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError)
	BulkDeleteCustomers(ctx context.Context, req *BulkDeleteCustomersRequest) (*apiresource.EmptyResource, *apierror.APIError)
	GetFrequentlyOrderedProducts(ctx context.Context, req *GetFrequentlyOrderedProductsRequest) (*apiresource.List[apiresource.FrequentlyOrderedProduct], *apierror.APIError)
	GetCustomerLeadTime(ctx context.Context, req *RetrieveCustomerLeadTimeRequest) (*apiresource.CustomerLeadTime, *apierror.APIError)
	ListNotificationRecipients(ctx context.Context, req *ListNotificationRecipientsRequest) (*apiresource.List[apiresource.OrderNotificationRecipient], *apierror.APIError)
	UpdateNotificationRecipients(ctx context.Context, req *UpdateNotificationRecipientsRequest) (*apiresource.List[apiresource.OrderNotificationRecipient], *apierror.APIError)
	MergeCustomers(ctx context.Context, req *MergeCustomersRequest) (*apiresource.Customer, *apierror.APIError)
	UpdateCustomer(ctx context.Context, req *UpdateCustomerRequest) (*apiresource.Customer, *apierror.APIError)
}

type CustomerSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
		Includes:              resourcekit.FilterIncludes(ctx, customerIncludes...),
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
		Includes: resourcekit.FilterIncludes(ctx, customerIncludes...),
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
	if v, ok := req.StatusCode.Value(); ok {
		statusCode = v
	}
	commissionPolicy := constants.CommissionPolicyExempt
	if v, ok := req.CommissionPolicy.Value(); ok {
		commissionPolicy = v
	}
	freightPolicy := constants.FreightPolicyBilled
	if v, ok := req.FreightPolicy.Value(); ok {
		freightPolicy = v
	}
	priorityCode := constants.PriorityCodeNormal
	if v, ok := req.DefaultPriorityCode.Value(); ok {
		priorityCode = v
	}

	statusCodeStr := string(statusCode)
	commissionPolicyStr := string(commissionPolicy)
	freightPolicyStr := string(freightPolicy)
	priorityCodeStr := string(priorityCode)

	pbReq := &pb.CreateCustomerRequest{
		Name:                  req.Name,
		Number:                req.Number.Ptr(),
		Note:                  req.Note.Ptr(),
		Email:                 req.Email.Ptr(),
		Phone:                 req.Phone.Ptr(),
		Url:                   req.URL.Ptr(),
		StatusCode:            &statusCodeStr,
		IsEdiEnabled:          ediStatusToBoolPtr(req.EDIStatus.Ptr()),
		CommissionPolicy:      &commissionPolicyStr,
		FreightPolicy:         &freightPolicyStr,
		DefaultLeadTimeDays:   req.LeadTimeDays.Ptr(),
		ReceiveCalendarId:     req.ReceiveCalendarID.Ptr(),
		FulfillmentPolicyCode: req.FulfillmentPolicy.Ptr().StringPtr(),
		DefaultCarrierId:      &req.DefaultCarrierID,
		DefaultServiceLevelId: req.DefaultServiceLevelID.Ptr(),
		DefaultPaymentTermId:  &req.DefaultPaymentTermID,
		DefaultShippingTermId: &req.DefaultShippingTermID,
		DefaultPriorityCode:   &priorityCodeStr,
		DefaultSalesRepId:     req.DefaultSalesRepID.Ptr(),
		CustomerPriceGroupIds: req.CustomerPriceGroupIDs,
		CustomerTypeGroupId:   &req.CustomerTypeGroupID,
		CarrierBillingType:    req.CarrierBillingType.Ptr().StringPtr(),
		CarrierBillingAccount: req.CarrierBillingAccount.Ptr(),
		Includes:              resourcekit.FilterIncludes(ctx, customerIncludes...),
	}

	if cl, ok := req.CreditLimit.Value(); ok {
		pbReq.CreditLimitValue = &cl.Value
		pbReq.CreditLimitUnitId = &cl.UnitID
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

	// The aggregation RPC only carries item IDs + counts. Hydrate the full item resources through the shared loader so the response matches their detail shape, instead of emitting partial stubs (which previously surfaced empty sku/type/timestamps and, worse, put the item description in the sku field). The item batch-get is counterparty-aware, so this works for the customer-portal relation actor.
	itemIDs := make([]string, 0, len(resp.Products))
	seenItem := make(map[string]struct{}, len(resp.Products))
	for _, p := range resp.Products {
		if _, ok := seenItem[p.ItemId]; !ok {
			seenItem[p.ItemId] = struct{}{}
			itemIDs = append(itemIDs, p.ItemId)
		}
	}

	itemsByID, apiErr := resourceloaders.LoadItems(ctx, itemIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	products := make([]apiresource.FrequentlyOrderedProduct, 0, len(resp.Products))
	for _, p := range resp.Products {
		item, ok := itemsByID[p.ItemId].(*apiresource.Item)
		if !ok || item == nil {
			// Item was deleted since it was last ordered; skip it rather than emit a stub row.
			continue
		}
		products = append(products, apiresource.FrequentlyOrderedProduct{
			Object:     constants.ObjectTypeFrequentlyOrderedProduct,
			Item:       item,
			OrderCount: p.OrderCount,
		})
	}

	return apiresource.NewList(products, apiresource.PageInfo{}), nil
}

func (m *customerSvcImpl) ListNotificationRecipients(ctx context.Context, req *ListNotificationRecipientsRequest) (*apiresource.List[apiresource.OrderNotificationRecipient], *apierror.APIError) {
	pbReq := &pb.ListCustomerNotificationRecipientsRequest{
		CustomerId: req.CustomerID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.list_notification_recipients", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCustomerNotificationRecipientsResponse, error) {
			return m.coreClient.ListCustomerNotificationRecipients(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return buildNotificationRecipients(ctx, resp.Recipients), nil
}

func (m *customerSvcImpl) UpdateNotificationRecipients(ctx context.Context, req *UpdateNotificationRecipientsRequest) (*apiresource.List[apiresource.OrderNotificationRecipient], *apierror.APIError) {
	recipients := make([]*pb.CustomerNotificationRecipientInputProto, len(req.Recipients))
	for i, r := range req.Recipients {
		codes := make([]string, len(r.NotificationTypes))
		for j, t := range r.NotificationTypes {
			codes[j] = string(t)
		}
		recipients[i] = &pb.CustomerNotificationRecipientInputProto{
			AccountUserId:         r.AccountUserID,
			NotificationTypeCodes: codes,
		}
	}

	pbReq := &pb.UpdateCustomerNotificationRecipientsRequest{
		CustomerId: req.CustomerID,
		Recipients: recipients,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.update_notification_recipients", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateCustomerNotificationRecipientsResponse, error) {
			return m.coreClient.UpdateCustomerNotificationRecipients(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return buildNotificationRecipients(ctx, resp.Recipients), nil
}

// buildNotificationRecipients maps the core-service recipients into the API resource.
// account_user is expandable and is hydrated inline only when ?include=account_user is
// requested. Hydration happens in core-service (the recipients belong to the customer's
// account, which the gateway's target-scoped account-user loader can't reach); the account
// user's name/email live on its `user` sub-resource, which is not exposed here because the
// global user loader is internal-only (portal relation actors can't use it) — clients resolve
// names from the customer's own account-users list instead.
func buildNotificationRecipients(ctx context.Context, protoRecipients []*pb.CustomerNotificationRecipientProto) *apiresource.List[apiresource.OrderNotificationRecipient] {
	includeAccountUser := resourcekit.RequestedIncludeSet(ctx)["account_user"]

	recipients := make([]apiresource.OrderNotificationRecipient, 0, len(protoRecipients))
	for _, r := range protoRecipients {
		notifTypes := make([]constants.AccountRelationNotificationType, 0, len(r.NotificationTypeCodes))
		for _, code := range r.NotificationTypeCodes {
			notifTypes = append(notifTypes, constants.AccountRelationNotificationType(code))
		}
		recipient := apiresource.OrderNotificationRecipient{
			Object:            constants.ObjectTypeOrderNotificationRecipient,
			NotificationTypes: notifTypes,
		}
		if includeAccountUser && r.AccountUser != nil {
			recipient.AccountUser = accountUserFromRecipientProto(r.AccountUser)
		}
		recipients = append(recipients, recipient)
	}

	return apiresource.NewList(recipients, apiresource.PageInfo{})
}

// accountUserFromRecipientProto builds the base AccountUser resource from the hydrated proto
// detail. Profile fields (name/email) live on the expandable `user` sub-resource and are left
// unset here; clients resolve them via the account-users list scoped to the customer's account.
func accountUserFromRecipientProto(au *pb.AccountUserDetail) *apiresource.AccountUser {
	return &apiresource.AccountUser{
		ID:         au.Id,
		Object:     constants.ObjectTypeAccountUser,
		Status:     constants.AccountUserStatus(au.StatusCode),
		LastUsedAt: grpcutil.TimestampToTimePtr(au.LastUsedAt),
		CreatedAt:  grpcutil.TimestampToTime(au.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(au.UpdatedAt),
	}
}

func (m *customerSvcImpl) MergeCustomers(ctx context.Context, req *MergeCustomersRequest) (*apiresource.Customer, *apierror.APIError) {
	pbReq := &pb.MergeCustomersRequest{
		TargetCustomerId:  req.CustomerID,
		SourceCustomerIds: req.SourceCustomerIDs,
		Includes:          resourcekit.FilterIncludes(ctx, customerIncludes...),
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
		Name:                     req.Name.Ptr(),
		Number:                   req.Number.Ptr(),
		Note:                     field.StringClearableToProto(req.Note),
		Email:                    field.StringClearableToProto(req.Email),
		Phone:                    field.StringClearableToProto(req.Phone),
		Url:                      field.StringClearableToProto(req.URL),
		StatusCode:               req.StatusCode.Ptr().StringPtr(),
		IsEdiEnabled:             ediStatusToBoolPtr(req.EDIStatus.Ptr()),
		CommissionPolicy:         req.CommissionPolicy.Ptr().StringPtr(),
		FreightPolicy:            req.FreightPolicy.Ptr().StringPtr(),
		DefaultLeadTimeDays:      field.Int32ClearableToProto(req.LeadTimeDays),
		ReceiveCalendarId:        field.StringClearableToProto(req.ReceiveCalendarID),
		FulfillmentPolicyCode:    field.EnumClearableToProto(req.FulfillmentPolicy),
		DefaultCarrierId:         req.DefaultCarrierID.Ptr(),
		DefaultServiceLevelId:    field.StringClearableToProto(req.DefaultServiceLevelID),
		DefaultPaymentTermId:     req.DefaultPaymentTermID.Ptr(),
		DefaultShippingTermId:    req.DefaultShippingTermID.Ptr(),
		DefaultPriorityCode:      req.DefaultPriorityCode.Ptr().StringPtr(),
		DefaultSalesRepId:        field.StringClearableToProto(req.DefaultSalesRepID),
		BillToAddressId:          field.StringClearableToProto(req.BillToAddressID),
		ShipToAddressId:          field.StringClearableToProto(req.ShipToAddressID),
		CustomerPriceGroupIds:    ptrutil.Deref(req.CustomerPriceGroupIDs.Ptr()),
		CustomerTypeGroupId:      req.CustomerTypeGroupID.Ptr(),
		CarrierBillingType:       req.CarrierBillingType.Ptr().StringPtr(),
		CarrierBillingAccount:    field.StringClearableToProto(req.CarrierBillingAccount),
		HasCustomerPriceGroupIds: req.CustomerPriceGroupIDs.IsSet(),
		Includes:                 resourcekit.FilterIncludes(ctx, customerIncludes...),
	}

	pbReq.CreditLimit = apirequest.QuantityFieldToProto(req.CreditLimit)

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
		IsDropShip:   addressTypeToDropShip(a.Type.Ptr()),
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
		// Stash only the FK id: the carrier sub fetches the full Carrier via LoadCarriers (so its
		// service_level_ids preview is available for freight_preferences.carrier.service_levels). Never fabricate.
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "fp_carrier_id", c.DefaultCarrier.Id)
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
	if c.DefaultSalesRep != nil {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "defaults_sales_rep_id", c.DefaultSalesRep.Id)
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
			// Unit left nil: stash the FK id so LoadUnits fetches the real Unit on
			// ?include=credit_limit.unit. Never fabricate.
		})
		if c.CreditLimit.UnitId != "" {
			meta.Set(constants.ObjectTypeQuantity, c.CreditLimit.Id, "unit_id", c.CreditLimit.UnitId)
		}
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

	// parent_account and child_accounts are expandable Customer references:
	// stash the FK ids so LoadCustomers fetches the real Customers on ?include=.
	// Never fabricate.
	if c.ParentAccount != nil && c.ParentAccount.Id != "" {
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "parent_account_id", c.ParentAccount.Id)
	}
	if len(c.ChildAccounts) > 0 {
		childIDs := make([]string, 0, len(c.ChildAccounts))
		for _, child := range c.ChildAccounts {
			if child.Id != "" {
				childIDs = append(childIDs, child.Id)
			}
		}
		meta.Set(constants.ObjectTypeCustomer, cust.ID, "child_account_ids", childIDs)
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
	d := &apiresource.CustomerDefaults{Object: constants.ObjectTypeCustomerDefaults, LeadTimeDays: c.DefaultLeadTimeDays, ReceiveCalendarID: c.ReceiveCalendarId}
	if c.FulfillmentPolicyCode != nil && *c.FulfillmentPolicyCode != "" {
		fp := constants.FulfillmentPolicy(*c.FulfillmentPolicyCode)
		d.FulfillmentPolicy = &fp
	}

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
		// Real priority data from the proto (id, code, name); owner is an
		// expandable sub-resource left nil. Never fabricate.
		d.Priority = &apiresource.Priority{
			ID:        c.DefaultPriority.Id,
			Object:    constants.ObjectTypePriority,
			Code:      constants.PriorityCode(c.DefaultPriority.Code),
			Name:      c.DefaultPriority.Name,
			CreatedAt: grpcutil.TimestampToTime(c.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(c.CreatedAt),
		}
	}

	// sales_rep is an expandable reference: the FK id is stashed in LoadMeta so
	// LoadAccountUsers fetches the real account user on
	// ?include=defaults.sales_rep. Never fabricate.

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

func (m *customerSvcImpl) GetCustomerLeadTime(ctx context.Context, req *RetrieveCustomerLeadTimeRequest) (*apiresource.CustomerLeadTime, *apierror.APIError) {
	pbReq := &pb.GetCustomerLeadTimeRequest{Id: req.CustomerID}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.get_lead_time", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCustomerLeadTimeResponse, error) {
			return m.coreClient.GetCustomerLeadTime(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := &apiresource.CustomerLeadTime{
		Object:   constants.ObjectTypeCustomerLeadTime,
		Customer: apiresource.NewEntity(resp.CustomerId, constants.ObjectTypeCustomer, nil, nil),
		Days:     resp.Days,
		Source:   constants.LeadTimeSource(resp.Source),
	}
	// The winning rule's id is loader-side metadata rather than a public field: the `account_group` and `parent_customer` includes resolve it into the nested resource, keyed by the customer the lead time was resolved for. Only one is ever set — whichever rule decided — so an include the source did not produce stays null.
	meta := resourcekit.GetLoadMeta(ctx)
	if resp.AccountGroupId != nil && *resp.AccountGroupId != "" {
		meta.Set(constants.ObjectTypeCustomerLeadTime, resp.CustomerId, "account_group_id", *resp.AccountGroupId)
	}
	if resp.ParentCustomerId != nil && *resp.ParentCustomerId != "" {
		meta.Set(constants.ObjectTypeCustomerLeadTime, resp.CustomerId, "parent_customer_id", *resp.ParentCustomerId)
	}
	return out, nil
}
