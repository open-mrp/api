package salesorderep

import (
	"context"
	"time"

	"github.com/open-mrp/apikit/include"
	"google.golang.org/grpc"

	itemep "github.com/open-mrp/api/services/api-gateway/endpoints/items"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	"github.com/open-mrp/api/shared/metadatapb"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/metadata"
	"github.com/open-mrp/apikit/timeutil"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (m *salesOrderSvcImpl) ListSalesOrders(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrder], *apierror.APIError) {
	if apiErr := req.CreatedRange.Validate(); apiErr != nil {
		return nil, apiErr
	}
	statusCodes := make([]string, len(req.Statuses))
	for i, s := range req.Statuses {
		statusCodes[i] = string(s.Code())
	}
	pbReq := &pb.ListSalesOrdersRequest{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		StatusCodes:      statusCodes,
		PaymentStatuses:  enumStrings(req.PaymentStatuses),
		ItemIds:          req.ItemIDs,
		ProductLineIds:   req.ProductLineIDs,
		CustomerIds:      req.CustomerIDs,
		CustomerGroupIds: req.GroupIDs,
		SalesRepIds:      req.SalesRepIDs,
		Created:          req.CreatedRange.ToProto(),
		ShipByAfter:      req.ShipByAfter,
		ShipByBefore:     req.ShipByBefore,
		PastDue:          req.PastDue,
		Includes:         include.FilterIncludes(ctx, salesOrderIncludeFields...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesOrdersResponse, error) {
			return m.coreClient.ListSalesOrders(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	orders := make([]apiresource.SalesOrder, len(resp.SalesOrders))
	for i, o := range resp.SalesOrders {
		orders[i] = SalesOrderFromProto(ctx, o)
	}
	return apiresource.NewList(orders, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *salesOrderSvcImpl) GetSalesOrder(ctx context.Context, req *RetrieveSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSalesOrderResponse, error) {
			return m.coreClient.GetSalesOrder(ctx, &pb.GetSalesOrderRequest{
				Id:       req.SalesOrderID,
				Includes: include.FilterIncludes(ctx, salesOrderIncludeFields...),
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return new(SalesOrderFromProto(ctx, resp.SalesOrder)), nil
}

func (m *salesOrderSvcImpl) CreateSalesOrder(ctx context.Context, req *CreateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	if currency, ok := req.Currency.Value(); ok && currency != string(constants.CurrencyUSD) {
		return nil, apperr.NewCurrencyNotSupportedError("currency", "Only `usd` is supported.")
	}

	lines := make([]*pb.CreateSalesOrderLineInput, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = &pb.CreateSalesOrderLineInput{
			Metadata:           metadata.FromCreateRequest(l.Metadata),
			ProductId:          l.ProductID,
			ProductSku:         l.ProductSKU.Ptr(),
			ProductDescription: l.ProductDescription.Ptr(),
			QuantityValue:      l.Quantity.Value,
			QuantityUnitId:     l.Quantity.UnitID,
			UnitPrice:          apirequest.OptionalUnitPriceToProto(l.UnitPrice),
		}
	}

	pbReq := &pb.CreateSalesOrderRequest{
		Metadata:         metadata.FromCreateRequest(req.Metadata),
		BuyerAccountId:   req.CustomerID,
		CustomerPoNumber: req.CustomerPurchaseOrderNumber.Ptr(),
		Instructions:     fieldpb.StringClearableToProto(req.Instructions),
		SiteId:           req.SiteID.Ptr(),
		SalesRepId:       req.SalesRepID.Ptr(),
		PaymentTermId:    req.PaymentTermID.Ptr(),
		ShippingTermId:   req.ShippingTermID.Ptr(),
		OrderDiscountId:  req.OrderDiscountID.Ptr(),
		Lines:            lines,
		Includes:         include.FilterIncludes(ctx, salesOrderIncludeFields...),
	}
	var apiErr *apierror.APIError
	if pbReq.BillTo, apiErr = snapshotInputToProto(req.BillTo, "bill_to"); apiErr != nil {
		return nil, apiErr
	}
	if pbReq.ShipTo, apiErr = snapshotInputToProto(req.ShipTo, "ship_to"); apiErr != nil {
		return nil, apiErr
	}
	if freight, ok := req.Freight.Value(); ok {
		pbReq.CarrierId = freight.CarrierID.Ptr()
		pbReq.ServiceLevelId = freight.ServiceLevelID.Ptr()
		if billing, ok := freight.CarrierBilling.Value(); ok {
			pbReq.CarrierBillingType = new(string(billing.Type))
			pbReq.CarrierBillingAccount = billing.AccountNumber.Ptr()
		}
	}
	if commitment, ok := req.Commitment.Value(); ok {
		if v, ok := commitment.PromisedOn.Value(); ok {
			pbReq.PromisedAt = timestamppb.New(v.Start(time.UTC))
		}
		if v, ok := commitment.ShipByOverrideOn.Value(); ok {
			pbReq.ShipByOverrideDate = timestamppb.New(v.Start(time.UTC))
		}
		if pbReq.LeadTimeOverrideDays, apiErr = apirequest.OptionalLeadTimeDays(commitment.LeadTimeOverride, "commitment.lead_time_override"); apiErr != nil {
			return nil, apiErr
		}
		pbReq.CommitmentFirmness = commitment.Firmness.Ptr().StringPtr()
	}
	if recipients, ok := req.EmailRecipients.Value(); ok {
		pbReq.EmailRecipients = apirequest.EmailRecipientsToProto(recipients)
		pbReq.HasEmailRecipients = true
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateSalesOrderResponse, error) {
			return m.coreClient.CreateSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return new(SalesOrderFromProto(ctx, resp.SalesOrder)), nil
}

func (m *salesOrderSvcImpl) UpdateSalesOrder(ctx context.Context, req *UpdateSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	pbReq := &pb.UpdateSalesOrderRequest{
		Metadata:         metadatapb.PatchToProto(req.Metadata),
		Id:               req.SalesOrderID,
		CustomerId:       req.CustomerID.Ptr(),
		CustomerPoNumber: fieldpb.StringClearableToProto(req.CustomerPurchaseOrderNumber),
		Instructions:     fieldpb.StringClearableToProto(req.Instructions),
		SiteId:           req.SiteID.Ptr(),
		SalesRepId:       fieldpb.StringClearableToProto(req.SalesRepID),
		PaymentTermId:    req.PaymentTermID.Ptr(),
		ShippingTermId:   req.ShippingTermID.Ptr(),
		OrderDiscountId:  fieldpb.StringClearableToProto(req.OrderDiscountID),
		Includes:         include.FilterIncludes(ctx, salesOrderIncludeFields...),
	}
	var apiErr *apierror.APIError
	if pbReq.BillTo, apiErr = snapshotInputToProto(req.BillTo, "bill_to"); apiErr != nil {
		return nil, apiErr
	}
	if pbReq.ShipTo, apiErr = snapshotInputToProto(req.ShipTo, "ship_to"); apiErr != nil {
		return nil, apiErr
	}
	if freight, ok := req.Freight.Value(); ok {
		pbReq.CarrierId = freight.CarrierID.Ptr()
		pbReq.ServiceLevelId = fieldpb.StringClearableToProto(freight.ServiceLevelID)
		switch {
		case freight.CarrierBilling.IsClear():
			pbReq.CarrierBillingType = &pb.StringPatch{Clear: true}
			pbReq.CarrierBillingAccount = &pb.StringPatch{Clear: true}
		case freight.CarrierBilling.IsSet():
			billing, _ := freight.CarrierBilling.Value()
			pbReq.CarrierBillingType = &pb.StringPatch{Value: new(string(billing.Type))}
			if account, ok := billing.AccountNumber.Value(); ok {
				pbReq.CarrierBillingAccount = &pb.StringPatch{Value: &account}
			} else {
				pbReq.CarrierBillingAccount = &pb.StringPatch{Clear: true}
			}
		}
	}
	if commitment, ok := req.Commitment.Value(); ok {
		pbReq.PromisedAt = dateClearableToProto(commitment.PromisedOn)
		pbReq.ShipByOverrideDate = dateClearableToProto(commitment.ShipByOverrideOn)
		if pbReq.LeadTimeOverrideDays, apiErr = apirequest.LeadTimeDaysPatch(commitment.LeadTimeOverride, "commitment.lead_time_override"); apiErr != nil {
			return nil, apiErr
		}
		switch {
		case commitment.Firmness.IsClear():
			pbReq.CommitmentFirmness = &pb.StringPatch{Clear: true}
		case commitment.Firmness.IsSet():
			v, _ := commitment.Firmness.Value()
			pbReq.CommitmentFirmness = &pb.StringPatch{Value: new(string(v))}
		}
	}
	if v, ok := req.AcknowledgmentStatus.Value(); ok {
		pbReq.IsAcknowledgmentSent = new(v == constants.AcknowledgmentStatusSent)
	}
	pbReq.AcknowledgementEmailContacts = emailContactListToProto(req.AcknowledgementEmailContacts)
	pbReq.InvoiceEmailContacts = emailContactListToProto(req.InvoiceEmailContacts)
	if recipients, ok := req.EmailRecipients.Value(); ok {
		pbReq.EmailRecipients = apirequest.EmailRecipientsToProto(recipients)
		pbReq.HasEmailRecipients = true
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateSalesOrderResponse, error) {
			return m.coreClient.UpdateSalesOrder(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return new(SalesOrderFromProto(ctx, resp.SalesOrder)), nil
}

// snapshotInputToProto is an address sent as param, as core-service takes it, once it is known to name a saved
// address or give one inline; nil when it was left out.
func snapshotInputToProto(f field.Optional[apirequest.AddressSnapshotInput], param string) (*pb.AddressSnapshot, *apierror.APIError) {
	in, ok := f.Value()
	if !ok {
		return nil, nil
	}
	if apiErr := in.Validate(param); apiErr != nil {
		return nil, apiErr
	}
	return in.ToProto(), nil
}

// dateClearableToProto is a clearable business day as the instant core-service stores it: the date at midnight UTC.
func dateClearableToProto(c field.Clearable[timeutil.Date]) *pb.TimestampPatch {
	switch {
	case c.IsClear():
		return &pb.TimestampPatch{Clear: true}
	case c.IsSet():
		v, _ := c.Value()
		return &pb.TimestampPatch{Value: timestamppb.New(v.Start(time.UTC))}
	}
	return nil
}

func (m *salesOrderSvcImpl) IssueSalesOrder(ctx context.Context, req *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeIssue, req.NotifyCustomer)
}

func (m *salesOrderSvcImpl) UnissueSalesOrder(ctx context.Context, req *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeUnissue, false)
}

func (m *salesOrderSvcImpl) CancelSalesOrder(ctx context.Context, req *CancelSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeCancel, false)
}

func (m *salesOrderSvcImpl) Preview9IssueSalesOrder(ctx context.Context, req *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	if apiErr := itemep.RequirePreview9(ctx, salesOrderStatusActionsRemoved); apiErr != nil {
		return nil, apiErr
	}
	return m.IssueSalesOrder(ctx, req)
}

func (m *salesOrderSvcImpl) Preview9UnissueSalesOrder(ctx context.Context, req *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	if apiErr := itemep.RequirePreview9(ctx, salesOrderStatusActionsRemoved); apiErr != nil {
		return nil, apiErr
	}
	return m.UnissueSalesOrder(ctx, req)
}

func (m *salesOrderSvcImpl) Preview9CloseSalesOrder(ctx context.Context, req *CancelSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	if apiErr := itemep.RequirePreview9(ctx, salesOrderStatusActionsRemoved); apiErr != nil {
		return nil, apiErr
	}
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeClose, false)
}

func (m *salesOrderSvcImpl) Preview9OpenSalesOrder(ctx context.Context, req *CancelSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
	if apiErr := itemep.RequirePreview9(ctx, salesOrderStatusActionsRemoved); apiErr != nil {
		return nil, apiErr
	}
	return m.changeSalesOrderStatus(ctx, req.SalesOrderID, constants.SalesOrderStatusChangeOpen, false)
}

// changeSalesOrderStatus runs one status transition and answers with the order.
func (m *salesOrderSvcImpl) changeSalesOrderStatus(ctx context.Context, id string, action constants.SalesOrderStatusChange, notifyCustomer bool) (*apiresource.SalesOrder, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderEpSvcTracer, "service.sales_orders.change_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangeSalesOrderStatusResponse, error) {
			return m.coreClient.ChangeSalesOrderStatus(ctx, &pb.ChangeSalesOrderStatusRequest{
				Id:           id,
				StatusChange: string(action),
				SendEmail:    notifyCustomer,
				Includes:     include.FilterIncludes(ctx, salesOrderIncludeFields...),
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return new(SalesOrderFromProto(ctx, resp.SalesOrder)), nil
}
