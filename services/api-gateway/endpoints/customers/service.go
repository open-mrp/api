package customerep

import (
	"context"
	"fmt"

	"github.com/open-mrp/apikit/include"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/ptrutil"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func standingsToStrings(codes []constants.CustomerStanding) []string {
	out := make([]string, len(codes))
	for i, c := range codes {
		out[i] = string(c)
	}
	return out
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
	MergeCustomers(ctx context.Context, req *MergeCustomersRequest) (*apiresource.AsyncJob, *apierror.APIError)
	UpdateCustomer(ctx context.Context, req *UpdateCustomerRequest) (*apiresource.Customer, *apierror.APIError)
	ExportCustomers(ctx context.Context, req *ExportCustomersRequest) (*apiresource.AsyncJob, *apierror.APIError)
}

type CustomerSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
	if apiErr := req.CreatedRange.Validate(); apiErr != nil {
		return nil, apiErr
	}
	pbReq := &pb.ListCustomersRequest{
		Cursor:                req.Cursor,
		Limit:                 req.Limit,
		Query:                 req.Query,
		CustomerGroupIds:      req.GroupIDs,
		PricingGroupIds:       req.PricingGroupIDs,
		PriceListIds:          req.PriceListIDs,
		SalesRepIds:           req.SalesRepIDs,
		StatusCodes:           standingsToStrings(req.Standings),
		ShippingTermIds:       req.ShippingTermIDs,
		PaymentTermIds:        req.PaymentTermIDs,
		CommissionStatusCodes: commissionPoliciesToStrings(req.CommissionPolicies),
		FreightStatusCodes:    freightPoliciesToStrings(req.FreightPolicies),
		CarrierIds:            req.CarrierIDs,
		ServiceLevelIds:       req.ServiceLevelIDs,
		City:                  req.City,
		State:                 req.State,
		PostalCode:            req.PostalCode,
		Created:               req.CreatedRange.ToProto(),
		Includes:              resourceloaders.CustomerCoreIncludes(ctx),
	}
	for _, s := range req.Statuses {
		pbReq.Statuses = append(pbReq.Statuses, string(s))
	}
	for _, t := range req.RelationshipTypes {
		pbReq.RelationshipTypes = append(pbReq.RelationshipTypes, string(t))
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
		items[i] = *resourceloaders.CustomerFromProto(ctx, c)
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *customerSvcImpl) ExportCustomers(ctx context.Context, req *ExportCustomersRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	pbReq := &pb.ExportCustomersRequest{
		Query:                 req.Query.Ptr(),
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
		IsParentAccount:       parentAccountStatusToBoolPtr(req.ParentAccountStatus.Ptr()),
		City:                  req.City.Ptr(),
		State:                 req.State.Ptr(),
		PostalCode:            req.PostalCode.Ptr(),
	}
	if startDate, ok := req.StartDate.Value(); ok {
		pbReq.StartDate = timestamppb.New(startDate)
	}
	if endDate, ok := req.EndDate.Value(); ok {
		pbReq.EndDate = timestamppb.New(endDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportCustomersResponse, error) {
			return m.coreClient.ExportCustomers(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}

func (m *customerSvcImpl) GetCustomer(ctx context.Context, req *RetrieveCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
	pbReq := &pb.GetCustomerRequest{
		Id:       req.CustomerID,
		Includes: resourceloaders.CustomerCoreIncludes(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCustomerResponse, error) {
			return m.coreClient.GetCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return resourceloaders.CustomerFromProto(ctx, resp.Customer), nil
}

func (m *customerSvcImpl) CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
	standing := string(fieldpb.ApplyOptional(constants.CustomerStandingNormal, req.Standing))
	commissionPolicy := string(fieldpb.ApplyOptional(constants.CommissionPolicyExempt, req.CommissionPolicy))
	freightPolicy := string(fieldpb.ApplyOptional(constants.FreightPolicyBilled, req.FreightPolicy))

	leadTime, apiErr := apirequest.OptionalLeadTimeDays(req.Defaults.LeadTime, "defaults.lead_time")
	if apiErr != nil {
		return nil, apiErr
	}

	pbReq := &pb.CreateCustomerRequest{
		Name:                  req.Name,
		Number:                req.Number.Ptr(),
		Instructions:          req.Instructions.Ptr(),
		StatusCode:            &standing,
		ArchiveStatus:         req.Status.Ptr().StringPtr(),
		CommissionPolicy:      &commissionPolicy,
		FreightPolicy:         &freightPolicy,
		DefaultLeadTimeDays:   leadTime,
		ReceiveCalendarId:     req.Defaults.ReceiveCalendarID.Ptr(),
		Supply:                req.Defaults.Supply.Ptr().StringPtr(),
		DefaultCarrierId:      &req.Defaults.CarrierID,
		DefaultServiceLevelId: req.Defaults.ServiceLevelID.Ptr(),
		DefaultPaymentTermId:  &req.Defaults.PaymentTermID,
		DefaultShippingTermId: &req.Defaults.ShippingTermID,
		DefaultSalesRepId:     req.Defaults.SalesRepID.Ptr(),
		CustomerPriceGroupIds: req.PriceGroupIDs,
		PriceListIds:          req.PriceListIDs,
		HasPriceListIds:       req.PriceListIDs != nil,
		CustomerTypeGroupId:   &req.GroupID,
		ParentCustomerId:      req.ParentCustomerID.Ptr(),
		Includes:              resourceloaders.CustomerCoreIncludes(ctx),
	}
	if contact, ok := req.Contact.Value(); ok {
		pbReq.Email, pbReq.Phone, pbReq.Url = contact.Email.ValuePtr(), contact.Phone.ValuePtr(), contact.URL.ValuePtr()
	}
	billingType := string(constants.CarrierBillingTypeSender)
	if billing, ok := req.CarrierBilling.Value(); ok {
		billingType = string(fieldpb.ApplyOptional(constants.CarrierBillingTypeSender, billing.Type))
		pbReq.CarrierBillingAccount = billing.AccountNumber.ValuePtr()
	}
	pbReq.CarrierBillingType = &billingType

	pbReq.CreditLimit = apirequest.OptionalMoneyToProto(req.CreditLimit)
	pbReq.BillToAddress = addressInputToCustomerProto(&req.BillToAddress)
	pbReq.ShipToAddress = addressInputToCustomerProto(&req.ShipToAddress)

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCustomerResponse, error) {
			return m.coreClient.CreateCustomer(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.CustomerFromProto(ctx, resp.Customer), nil
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
	includeAccountUser := include.RequestedIncludeSet(ctx)["account_user"]

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

func (m *customerSvcImpl) MergeCustomers(ctx context.Context, req *MergeCustomersRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	pbReq := &pb.MergeCustomersRequest{
		TargetCustomerId:  req.CustomerID,
		SourceCustomerIds: req.SourceCustomerIDs,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.merge", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.MergeCustomersResponse, error) {
			return m.coreClient.MergeCustomers(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}

func (m *customerSvcImpl) UpdateCustomer(ctx context.Context, req *UpdateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
	pbReq := &pb.UpdateCustomerRequest{
		Id:                       req.CustomerID,
		Name:                     req.Name.Ptr(),
		Number:                   req.Number.Ptr(),
		Instructions:             fieldpb.StringClearableToProto(req.Instructions),
		StatusCode:               req.Standing.Ptr().StringPtr(),
		ArchiveStatus:            req.Status.Ptr().StringPtr(),
		CommissionPolicy:         req.CommissionPolicy.Ptr().StringPtr(),
		FreightPolicy:            req.FreightPolicy.Ptr().StringPtr(),
		CustomerPriceGroupIds:    ptrutil.Deref(req.PriceGroupIDs.Ptr()),
		HasCustomerPriceGroupIds: req.PriceGroupIDs.IsSet(),
		PriceListIds:             ptrutil.Deref(req.PriceListIDs.Ptr()),
		HasPriceListIds:          req.PriceListIDs.IsSet(),
		CustomerTypeGroupId:      req.GroupID.Ptr(),
		ParentCustomerId:         fieldpb.StringClearableToProto(req.ParentCustomerID),
		Includes:                 resourceloaders.CustomerCoreIncludes(ctx),
	}
	if contact, ok := req.Contact.Value(); ok {
		pbReq.Email = fieldpb.StringClearableToProto(contact.Email)
		pbReq.Phone = fieldpb.StringClearableToProto(contact.Phone)
		pbReq.Url = fieldpb.StringClearableToProto(contact.URL)
	}
	if billing, ok := req.CarrierBilling.Value(); ok {
		pbReq.CarrierBillingType = billing.Type.Ptr().StringPtr()
		pbReq.CarrierBillingAccount = fieldpb.StringClearableToProto(billing.AccountNumber)
	}
	if defaults, ok := req.Defaults.Value(); ok {
		leadTime, apiErr := apirequest.LeadTimeDaysPatch(defaults.LeadTime, "defaults.lead_time")
		if apiErr != nil {
			return nil, apiErr
		}
		pbReq.DefaultLeadTimeDays = leadTime
		pbReq.DefaultCarrierId = defaults.CarrierID.Ptr()
		pbReq.DefaultServiceLevelId = fieldpb.StringClearableToProto(defaults.ServiceLevelID)
		pbReq.DefaultPaymentTermId = defaults.PaymentTermID.Ptr()
		pbReq.DefaultShippingTermId = defaults.ShippingTermID.Ptr()
		pbReq.DefaultSalesRepId = fieldpb.StringClearableToProto(defaults.SalesRepID)
		pbReq.BillToAddressId = fieldpb.StringClearableToProto(defaults.BillToAddressID)
		pbReq.ShipToAddressId = fieldpb.StringClearableToProto(defaults.ShipToAddressID)
		pbReq.ReceiveCalendarId = fieldpb.StringClearableToProto(defaults.ReceiveCalendarID)
		pbReq.Supply = fieldpb.EnumClearableToProto(defaults.Supply)
	}
	if recipients, ok := req.NotificationRecipients.Value(); ok {
		pbReq.NotificationRecipients = recipientsToProto(recipients)
		pbReq.HasNotificationRecipients = true
	}
	pbReq.CreditLimit = apirequest.MoneyFieldToProto(req.CreditLimit)

	resp, apiErr := grpcutil.CallRPC(ctx, customerSvcTracer, "service.customers.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateCustomerResponse, error) {
			return m.coreClient.UpdateCustomer(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.CustomerFromProto(ctx, resp.Customer), nil
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
		StreetLine_1: a.Line1.Ptr(),
		StreetLine_2: a.Line2.Ptr(),
		Locality:     a.City.Ptr(),
		State:        a.State.Ptr(),
		PostalCode:   a.PostalCode.Ptr(),
		Country:      a.Country,
	}
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
		LeadTime: resourceloaders.DaysQuantity(&resp.Days),
		Source:   constants.LeadTimeSource(resp.Source),
	}
	// The winning rule's id is loader-side metadata rather than a public field: the `account_group` and `parent_customer` includes resolve it into the nested resource, keyed by the customer the lead time was resolved for. Only one is ever set — whichever rule decided — so an include the source did not produce stays null.
	meta := include.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeCustomerLeadTime, "lead_time", "customer_id", resp.CustomerId)
	if resp.AccountGroupId != nil && *resp.AccountGroupId != "" {
		meta.Set(constants.ObjectTypeCustomerLeadTime, "lead_time", "account_group_id", *resp.AccountGroupId)
	}
	if resp.ParentCustomerId != nil && *resp.ParentCustomerId != "" {
		meta.Set(constants.ObjectTypeCustomerLeadTime, "lead_time", "parent_customer_id", *resp.ParentCustomerId)
	}
	return out, nil
}
