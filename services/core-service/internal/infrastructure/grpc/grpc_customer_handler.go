package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func optStringToCommissionPolicy(s *string) *constants.CommissionPolicy {
	if s == nil {
		return nil
	}
	v := constants.CommissionPolicy(*s)
	return &v
}

func optStringToFreightPolicy(s *string) *constants.FreightPolicy {
	if s == nil {
		return nil
	}
	v := constants.FreightPolicy(*s)
	return &v
}

func customerAddressToProto(addr *domain.CustomerAddress) *pb.CustomerAddressProto {
	if addr == nil {
		return nil
	}

	p := &pb.CustomerAddressProto{
		Id:         addr.ID,
		Name:       addr.Name,
		Phone:      addr.Phone,
		Email:      addr.Email,
		IsDropShip: addr.IsDropShip,
		CreatedAt:  timestamppb.New(addr.CreatedAt),
		UpdatedAt:  timestamppb.New(addr.UpdatedAt),
	}

	if addr.Geolocation != nil {
		p.Geolocation = &pb.CustomerGeolocationProto{
			Id:           addr.Geolocation.ID,
			StreetLine_1: addr.Geolocation.StreetLine1,
			StreetLine_2: addr.Geolocation.StreetLine2,
			Locality:     addr.Geolocation.Locality,
			State:        addr.Geolocation.State,
			PostalCode:   addr.Geolocation.PostalCode,
			Country:      addr.Geolocation.Country,
		}
	}

	return p
}

func customerToProto(c *domain.Customer) *pb.CustomerProto {
	p := &pb.CustomerProto{
		Id:                    c.ID,
		Name:                  c.Name,
		Number:                c.Number,
		Status:                string(c.Status),
		IsEdiEnabled:          c.IsEdiEnabled,
		IsParentAccount:       c.IsParentAccount,
		CommissionPolicy:      string(c.CommissionPolicy),
		FreightPolicy:         string(c.FreightPolicy),
		Note:                  c.Note,
		Email:                 c.Email,
		Phone:                 c.Phone,
		Url:                   c.URL,
		CarrierBillingAccount: c.CarrierBillingAccount,
		AcceptsInvoiceEmails:  c.AcceptsInvoiceEmails,
		BillToAddress:         customerAddressToProto(c.BillToAddress),
		ShipToAddress:         customerAddressToProto(c.ShipToAddress),
		CreatedAt:             timestamppb.New(c.CreatedAt),
		UpdatedAt:             timestamppb.New(c.UpdatedAt),
	}

	if c.CarrierBillingType != nil {
		s := string(*c.CarrierBillingType)
		p.CarrierBillingType = &s
	}

	if c.CreditLimitID != nil {
		p.CreditLimit = &pb.CustomerCreditLimitProto{
			Id:               *c.CreditLimitID,
			Value:            *c.CreditLimitValue,
			UnitId:           *c.CreditLimitUnitID,
			UnitAbbreviation: *c.CreditLimitUnitAbbreviation,
			UnitName:         *c.CreditLimitUnitName,
			UnitType:         *c.CreditLimitUnitType,
		}
	}

	if c.DefaultCarrierID != nil {
		carrier := &pb.CustomerCarrierProto{
			Id:   *c.DefaultCarrierID,
			Name: *c.DefaultCarrierName,
		}
		if c.DefaultCarrierIsPortalEnabled != nil {
			carrier.IsPortalEnabled = *c.DefaultCarrierIsPortalEnabled
		}
		if c.DefaultCarrierCreatedAt != nil {
			carrier.CreatedAt = timestamppb.New(*c.DefaultCarrierCreatedAt)
		}
		if c.DefaultCarrierUpdatedAt != nil {
			carrier.UpdatedAt = timestamppb.New(*c.DefaultCarrierUpdatedAt)
		}
		p.DefaultCarrier = carrier
	}

	if c.DefaultServiceLevelID != nil {
		sl := &pb.CustomerServiceLevelProto{
			Id:   *c.DefaultServiceLevelID,
			Name: *c.DefaultServiceLevelName,
		}
		if c.DefaultServiceLevelToken != nil {
			sl.ServiceLevelToken = c.DefaultServiceLevelToken
		}
		if c.DefaultServiceLevelIsPortalEnabled != nil {
			sl.IsPortalEnabled = c.DefaultServiceLevelIsPortalEnabled
		}
		if c.DefaultServiceLevelCreatedAt != nil {
			sl.CreatedAt = timestamppb.New(*c.DefaultServiceLevelCreatedAt)
		}
		if c.DefaultServiceLevelUpdatedAt != nil {
			sl.UpdatedAt = timestamppb.New(*c.DefaultServiceLevelUpdatedAt)
		}
		p.DefaultServiceLevel = sl
	}

	if c.DefaultPaymentTermID != nil {
		pt := &pb.CustomerPaymentTermProto{
			Id:   *c.DefaultPaymentTermID,
			Name: *c.DefaultPaymentTermName,
		}
		if c.DefaultPaymentTermIsActive != nil {
			pt.IsActive = *c.DefaultPaymentTermIsActive
		}
		if c.DefaultPaymentTermCreatedAt != nil {
			pt.CreatedAt = timestamppb.New(*c.DefaultPaymentTermCreatedAt)
		}
		if c.DefaultPaymentTermUpdatedAt != nil {
			pt.UpdatedAt = timestamppb.New(*c.DefaultPaymentTermUpdatedAt)
		}
		p.DefaultPaymentTerm = pt
	}

	if c.DefaultShippingTermID != nil {
		st := &pb.CustomerShippingTermProto{
			Id:   *c.DefaultShippingTermID,
			Name: *c.DefaultShippingTermName,
		}
		if c.DefaultShippingTermType != nil {
			st.Type = string(*c.DefaultShippingTermType)
		}
		if c.DefaultShippingTermCreatedAt != nil {
			st.CreatedAt = timestamppb.New(*c.DefaultShippingTermCreatedAt)
		}
		if c.DefaultShippingTermUpdatedAt != nil {
			st.UpdatedAt = timestamppb.New(*c.DefaultShippingTermUpdatedAt)
		}
		p.DefaultShippingTerm = st
	}

	if c.DefaultPriorityCode != nil {
		priority := &pb.CustomerPriorityProto{
			Code: string(*c.DefaultPriorityCode),
			Name: *c.DefaultPriorityName,
		}
		if c.DefaultPriorityID != nil {
			priority.Id = *c.DefaultPriorityID
		}
		p.DefaultPriority = priority
	}

	if c.DefaultSalesRepID != nil && *c.DefaultSalesRepID != "" {
		sr := &pb.CustomerUserProto{
			Id:   *c.DefaultSalesRepID,
			Name: c.DefaultSalesRepName,
		}
		if c.DefaultSalesRepStatus != nil {
			s := string(*c.DefaultSalesRepStatus)
			sr.Status = &s
		}
		if c.DefaultSalesRepCreatedAt != nil {
			sr.CreatedAt = timestamppb.New(*c.DefaultSalesRepCreatedAt)
		}
		if c.DefaultSalesRepUpdatedAt != nil {
			sr.UpdatedAt = timestamppb.New(*c.DefaultSalesRepUpdatedAt)
		}
		p.DefaultSalesRep = sr
	}

	if c.TypeGroupID != nil {
		tg := &pb.CustomerAccountGroupProto{
			Id:   *c.TypeGroupID,
			Name: *c.TypeGroupName,
		}
		if c.TypeGroupCommissionPolicy != nil {
			tg.CommissionPolicy = string(*c.TypeGroupCommissionPolicy)
		}
		if c.TypeGroupFreightPolicy != nil {
			tg.FreightPolicy = string(*c.TypeGroupFreightPolicy)
		}
		if c.TypeGroupType != nil {
			tg.Type = string(*c.TypeGroupType)
		}
		if c.TypeGroupCreatedAt != nil {
			tg.CreatedAt = timestamppb.New(*c.TypeGroupCreatedAt)
		}
		if c.TypeGroupUpdatedAt != nil {
			tg.UpdatedAt = timestamppb.New(*c.TypeGroupUpdatedAt)
		}
		p.TypeGroup = tg
	}

	priceGroups := make([]*pb.CustomerAccountGroupProto, len(c.PriceGroups))
	for i, pg := range c.PriceGroups {
		priceGroups[i] = &pb.CustomerAccountGroupProto{
			Id:               pg.ID,
			Name:             pg.Name,
			CommissionPolicy: string(pg.CommissionPolicy),
			FreightPolicy:    string(pg.FreightPolicy),
			Type:             string(pg.Type),
			CreatedAt:        timestamppb.New(pg.CreatedAt),
			UpdatedAt:        timestamppb.New(pg.UpdatedAt),
		}
	}
	p.PriceGroups = priceGroups

	if c.ParentAccountID != nil {
		pa := &pb.CustomerLightCustomerProto{
			Id:     *c.ParentAccountID,
			Name:   *c.ParentAccountName,
			Number: *c.ParentAccountNumber,
		}
		if c.ParentAccountCreatedAt != nil {
			pa.CreatedAt = timestamppb.New(*c.ParentAccountCreatedAt)
		}
		if c.ParentAccountUpdatedAt != nil {
			pa.UpdatedAt = timestamppb.New(*c.ParentAccountUpdatedAt)
		}
		p.ParentAccount = pa
	}

	if len(c.ChildAccounts) > 0 {
		p.ChildAccounts = make([]*pb.CustomerLightCustomerProto, len(c.ChildAccounts))
		for i, child := range c.ChildAccounts {
			p.ChildAccounts[i] = &pb.CustomerLightCustomerProto{
				Id:        child.ID,
				Name:      child.Name,
				Number:    child.Number,
				CreatedAt: timestamppb.New(child.CreatedAt),
				UpdatedAt: timestamppb.New(child.UpdatedAt),
			}
		}
	}

	return p
}

func (h *gRPCHandler) ListCustomers(ctx context.Context, req *pb.ListCustomersRequest) (*pb.ListCustomersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListCustomersParams{
		Limit: req.Limit,
	}

	if req.Cursor != nil {
		params.Cursor = req.Cursor
	}
	if req.Query != nil {
		params.Query = req.Query
	}
	if req.IsParentAccount != nil {
		params.IsParentAccount = req.IsParentAccount
	}
	if req.City != nil {
		params.City = req.City
	}
	if req.State != nil {
		params.State = req.State
	}
	if req.PostalCode != nil {
		params.PostalCode = req.PostalCode
	}
	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	params.CustomerGroupIDs = req.CustomerGroupIds
	params.PricingGroupIDs = req.PricingGroupIds
	params.SalesRepIDs = req.SalesRepIds
	params.StatusCodes = req.StatusCodes
	params.ShippingTermIDs = req.ShippingTermIds
	params.PaymentTermIDs = req.PaymentTermIds
	params.CommissionPolicyCodes = req.CommissionStatusCodes
	params.FreightPolicyCodes = req.FreightStatusCodes
	params.CarrierIDs = req.CarrierIds
	params.ServiceLevelIDs = req.ServiceLevelIds
	params.Includes = req.Includes

	result, apiErr := h.customerSvc.ListCustomers(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	customers := make([]*pb.CustomerProto, len(result.Items))
	for i, c := range result.Items {
		customers[i] = customerToProto(c)
	}

	return &pb.ListCustomersResponse{
		Customers: customers,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetCustomer(ctx context.Context, req *pb.GetCustomerRequest) (*pb.GetCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	customer, apiErr := h.customerSvc.GetCustomer(ctx, req.Id, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetCustomerResponse{
		Customer: customerToProto(customer),
	}, nil
}

// BatchGetCustomersByIDs returns customers by ID for the api-gateway include resolver. It reuses the authorized single-get path per id; ids the caller cannot access or that no longer exist are omitted so the resolver leaves those references null.
func (h *gRPCHandler) BatchGetCustomersByIDs(ctx context.Context, req *pb.BatchGetCustomersByIDsRequest) (*pb.BatchGetCustomersByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	customers := make([]*pb.CustomerProto, 0, len(req.Ids))
	for _, id := range req.Ids {
		if id == "" {
			continue
		}
		customer, apiErr := h.customerSvc.GetCustomer(ctx, id, nil)
		if apiErr != nil {
			continue
		}
		customers = append(customers, customerToProto(customer))
	}

	return &pb.BatchGetCustomersByIDsResponse{Customers: customers}, nil
}

func (h *gRPCHandler) CreateCustomer(ctx context.Context, req *pb.CreateCustomerRequest) (*pb.CreateCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateCustomerParams{
		Name:                  req.Name,
		Number:                req.Number,
		Note:                  req.Note,
		Email:                 req.Email,
		Phone:                 req.Phone,
		URL:                   req.Url,
		StatusCode:            req.StatusCode,
		IsEdiEnabled:          req.IsEdiEnabled,
		CommissionPolicy:      optStringToCommissionPolicy(req.CommissionPolicy),
		FreightPolicy:         optStringToFreightPolicy(req.FreightPolicy),
		DefaultCarrierID:      req.DefaultCarrierId,
		DefaultServiceLevelID: req.DefaultServiceLevelId,
		DefaultPaymentTermID:  req.DefaultPaymentTermId,
		DefaultShippingTermID: req.DefaultShippingTermId,
		DefaultPriorityCode:   req.DefaultPriorityCode,
		DefaultSalesRepID:     req.DefaultSalesRepId,
		CustomerPriceGroupIDs: req.CustomerPriceGroupIds,
		CustomerTypeGroupID:   req.CustomerTypeGroupId,
		CarrierBillingType:    req.CarrierBillingType,
		CarrierBillingAccount: req.CarrierBillingAccount,
		CreditLimitValue:      req.CreditLimitValue,
		CreditLimitUnitID:     req.CreditLimitUnitId,
		Includes:              req.Includes,
	}

	if req.BillToAddress != nil {
		params.BillToAddress = protoCustomerAddressInputToCreateParams(req.BillToAddress)
	}
	if req.ShipToAddress != nil {
		params.ShipToAddress = protoCustomerAddressInputToCreateParams(req.ShipToAddress)
	}

	customer, apiErr := h.customerSvc.CreateCustomer(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateCustomerResponse{
		Customer: customerToProto(customer),
	}, nil
}

func (h *gRPCHandler) UpdateCustomer(ctx context.Context, req *pb.UpdateCustomerRequest) (*pb.UpdateCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateCustomerParams{
		CustomerAccountID:        req.Id,
		Name:                     req.Name,
		Number:                   req.Number,
		Note:                     field.StringClearableFromProto(req.Note),
		Email:                    field.StringClearableFromProto(req.Email),
		Phone:                    field.StringClearableFromProto(req.Phone),
		URL:                      field.StringClearableFromProto(req.Url),
		StatusCode:               req.StatusCode,
		IsEdiEnabled:             req.IsEdiEnabled,
		CommissionPolicy:         optStringToCommissionPolicy(req.CommissionPolicy),
		FreightPolicy:            optStringToFreightPolicy(req.FreightPolicy),
		DefaultCarrierID:         req.DefaultCarrierId,
		DefaultServiceLevelID:    field.StringClearableFromProto(req.DefaultServiceLevelId),
		DefaultPaymentTermID:     req.DefaultPaymentTermId,
		DefaultShippingTermID:    req.DefaultShippingTermId,
		DefaultPriorityCode:      req.DefaultPriorityCode,
		DefaultSalesRepID:        field.StringClearableFromProto(req.DefaultSalesRepId),
		BillToAddressID:          field.StringClearableFromProto(req.BillToAddressId),
		ShipToAddressID:          field.StringClearableFromProto(req.ShipToAddressId),
		CustomerPriceGroupIDs:    req.CustomerPriceGroupIds,
		HasCustomerPriceGroupIDs: req.HasCustomerPriceGroupIds,
		CustomerTypeGroupID:      req.CustomerTypeGroupId,
		CarrierBillingType:       req.CarrierBillingType,
		CarrierBillingAccount:    field.StringClearableFromProto(req.CarrierBillingAccount),
		CreditLimit:              field.QuantityClearableFromProto(req.CreditLimit),
		Includes:                 req.Includes,
	}

	customer, apiErr := h.customerSvc.UpdateCustomer(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateCustomerResponse{
		Customer: customerToProto(customer),
	}, nil
}

func (h *gRPCHandler) DeleteCustomer(ctx context.Context, req *pb.DeleteCustomerRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.customerSvc.DeleteCustomer(ctx, domain.DeleteCustomerParams{
		CustomerAccountID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) BulkDeleteCustomers(ctx context.Context, req *pb.BulkDeleteCustomersRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.customerSvc.BulkDeleteCustomers(ctx, domain.BulkDeleteCustomersParams{
		CustomerIDs: req.CustomerIds,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func frequentlyOrderedProductToProto(p *domain.FrequentlyOrderedProduct) *pb.FrequentlyOrderedProductProto {
	return &pb.FrequentlyOrderedProductProto{
		ItemId:           p.ItemID,
		ProductName:      p.ProductName,
		UnitId:           p.UnitID,
		UnitAbbreviation: p.UnitAbbreviation,
		OrderCount:       p.OrderCount,
	}
}

func (h *gRPCHandler) GetFrequentlyOrderedProducts(ctx context.Context, req *pb.GetFrequentlyOrderedProductsRequest) (*pb.GetFrequentlyOrderedProductsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	products, apiErr := h.customerSvc.GetFrequentlyOrderedProducts(ctx, req.CustomerId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbProducts := make([]*pb.FrequentlyOrderedProductProto, len(products))
	for i, p := range products {
		pbProducts[i] = frequentlyOrderedProductToProto(p)
	}

	return &pb.GetFrequentlyOrderedProductsResponse{
		Products: pbProducts,
	}, nil
}

func (h *gRPCHandler) ListCustomerNotificationRecipients(ctx context.Context, req *pb.ListCustomerNotificationRecipientsRequest) (*pb.ListCustomerNotificationRecipientsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	recipients, apiErr := h.customerSvc.ListCustomerNotificationRecipients(ctx, req.CustomerId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ListCustomerNotificationRecipientsResponse{
		Recipients: notificationRecipientsToProto(recipients),
	}, nil
}

func (h *gRPCHandler) UpdateCustomerNotificationRecipients(ctx context.Context, req *pb.UpdateCustomerNotificationRecipientsRequest) (*pb.UpdateCustomerNotificationRecipientsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	recipients := make([]domain.NotificationRecipientInput, len(req.Recipients))
	for i, r := range req.Recipients {
		recipients[i] = domain.NotificationRecipientInput{
			AccountUserID:         r.AccountUserId,
			NotificationTypeCodes: r.NotificationTypeCodes,
		}
	}

	updated, apiErr := h.customerSvc.UpdateCustomerNotificationRecipients(ctx, domain.UpdateCustomerNotificationRecipientsParams{
		CustomerAccountID: req.CustomerId,
		Recipients:        recipients,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateCustomerNotificationRecipientsResponse{
		Recipients: notificationRecipientsToProto(updated),
	}, nil
}

func notificationRecipientsToProto(recipients []domain.NotificationRecipient) []*pb.CustomerNotificationRecipientProto {
	out := make([]*pb.CustomerNotificationRecipientProto, len(recipients))
	for i, r := range recipients {
		out[i] = &pb.CustomerNotificationRecipientProto{
			AccountUser:           accountUserDetailToProto(r.AccountUser),
			NotificationTypeCodes: r.NotificationTypeCodes,
		}
	}
	return out
}

func (h *gRPCHandler) MergeCustomers(ctx context.Context, req *pb.MergeCustomersRequest) (*pb.MergeCustomersResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	customer, apiErr := h.customerSvc.MergeCustomers(ctx, domain.MergeCustomersParams{
		TargetCustomerID:  req.TargetCustomerId,
		SourceCustomerIDs: req.SourceCustomerIds,
		Includes:          req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.MergeCustomersResponse{
		Customer: customerToProto(customer),
	}, nil
}

func protoCustomerAddressInputToCreateParams(input *pb.CreateCustomerAddressInput) *domain.CreateAddressParams {
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
