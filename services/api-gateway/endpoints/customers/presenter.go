package customerep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func CustomerPresenter(c *pb.CustomerProto) apiresource.Customer {
	if c == nil {
		return apiresource.Customer{}
	}

	// Contact info — always populate so the include system can collapse it.
	contactInfo := &apiresource.CustomerContactInfo{
		Object: constants.ObjectTypeCustomerContactInfo,
		Email:  c.Email,
		Phone:  c.Phone,
		URL:    c.Url,
	}

	// Freight preferences
	var freightPreferences *apiresource.CustomerFreightPreferences
	{
		fp := &apiresource.CustomerFreightPreferences{
			Object: constants.ObjectTypeCustomerFreightPreferences,
			Status: constants.FreightPolicy(c.FreightPolicy),
		}

		if c.DefaultCarrier != nil {
			carrierVisibility := constants.CustomerPortalVisibilityHidden
			if c.DefaultCarrier.IsPortalEnabled {
				carrierVisibility = constants.CustomerPortalVisibilityVisible
			}
			fp.Carrier = &apiresource.Carrier{
				ID:                       c.DefaultCarrier.Id,
				Object:                   constants.ObjectTypeCarrier,
				Name:                     c.DefaultCarrier.Name,
				CustomerPortalVisibility: carrierVisibility,
			}
		}

		if c.DefaultServiceLevel != nil {
			fp.ServiceLevel = &apiresource.ServiceLevel{
				ID:     c.DefaultServiceLevel.Id,
				Object: constants.ObjectTypeServiceLevel,
				Name:   c.DefaultServiceLevel.Name,
			}
		}

		if c.CarrierBillingType != nil {
			bt := constants.CarrierBillingType(*c.CarrierBillingType)
			fp.BillingType = &bt
		}

		fp.BillingAccount = c.CarrierBillingAccount

		freightPreferences = fp
	}

	// Defaults
	var defaults *apiresource.CustomerDefaults
	{
		d := &apiresource.CustomerDefaults{Object: constants.ObjectTypeCustomerDefaults}

		if c.DefaultPaymentTerm != nil {
			ptStatus := constants.PaymentTermStatusInactive
			if c.DefaultPaymentTerm.IsActive {
				ptStatus = constants.PaymentTermStatusActive
			}
			d.PaymentTerm = &apiresource.PaymentTerm{
				ID:     c.DefaultPaymentTerm.Id,
				Object: constants.ObjectTypePaymentTerm,
				Name:   c.DefaultPaymentTerm.Name,
				Status: ptStatus,
			}
		}

		if c.DefaultShippingTerm != nil {
			d.ShippingTerm = &apiresource.ShippingTerm{
				ID:     c.DefaultShippingTerm.Id,
				Object: constants.ObjectTypeShippingTerm,
				Name:   c.DefaultShippingTerm.Name,
				Type:   constants.ShippingTermType(c.DefaultShippingTerm.Type),
			}
		}

		if c.DefaultPriority != nil {
			d.Priority = &apiresource.Priority{
				ID:     c.DefaultPriority.Id,
				Code:   constants.PriorityCode(c.DefaultPriority.Code),
				Object: constants.ObjectTypePriority,
				Name:   c.DefaultPriority.Name,
			}
		}

		if c.DefaultSalesRep != nil {
			d.SalesRep = &apiresource.User{
				ID:     c.DefaultSalesRep.Id,
				Object: constants.ObjectTypeUser,
				Name:   c.DefaultSalesRep.Name,
			}
		}

		defaults = d
	}

	// Notification preferences
	notificationPreferences := &apiresource.CustomerNotificationPreferences{
		Object:               constants.ObjectTypeCustomerNotificationPreferences,
		AcceptsInvoiceEmails: c.AcceptsInvoiceEmails,
	}

	// Bill-to address
	var billToAddress *apiresource.Address
	if c.BillToAddress != nil {
		billToAddress = customerAddressPresenter(c.BillToAddress)
	}

	// Ship-to address
	var shipToAddress *apiresource.Address
	if c.ShipToAddress != nil {
		shipToAddress = customerAddressPresenter(c.ShipToAddress)
	}

	// Type group
	var typeGroup *apiresource.AccountGroup
	if c.TypeGroup != nil {
		typeGroup = &apiresource.AccountGroup{
			ID:               c.TypeGroup.Id,
			Object:           constants.ObjectTypeAccountGroup,
			Name:             c.TypeGroup.Name,
			CommissionPolicy: constants.CommissionPolicy(c.TypeGroup.CommissionPolicy),
			FreightPolicy:    constants.FreightPolicy(c.TypeGroup.FreightPolicy),
			Type:             constants.AccountGroupType(c.TypeGroup.Type),
		}
	}

	// Price groups
	var priceGroups []apiresource.AccountGroup
	for _, pg := range c.PriceGroups {
		priceGroups = append(priceGroups, apiresource.AccountGroup{
			ID:               pg.Id,
			Object:           constants.ObjectTypeAccountGroup,
			Name:             pg.Name,
			CommissionPolicy: constants.CommissionPolicy(pg.CommissionPolicy),
			FreightPolicy:    constants.FreightPolicy(pg.FreightPolicy),
			Type:             constants.AccountGroupType(pg.Type),
		})
	}

	// Parent account
	var parentAccount *apiresource.Customer
	if c.ParentAccount != nil {
		parentAccount = &apiresource.Customer{
			ID:     c.ParentAccount.Id,
			Object: constants.ObjectTypeCustomer,
			Name:   c.ParentAccount.Name,
			Number: c.ParentAccount.Number,
		}
	}

	return apiresource.Customer{
		ID:                      c.Id,
		Object:                  constants.ObjectTypeCustomer,
		Name:                    c.Name,
		Number:                  c.Number,
		Status:                  constants.AccountStatusCode(c.Status),
		IsEdiEnabled:            c.IsEdiEnabled,
		IsParentAccount:         c.IsParentAccount,
		CommissionPolicy:        constants.CommissionPolicy(c.CommissionPolicy),
		Note:                    c.Note,
		ContactInfo:             contactInfo,
		FreightPreferences:      freightPreferences,
		Defaults:                defaults,
		NotificationPreferences: notificationPreferences,
		BillToAddress:           billToAddress,
		ShipToAddress:           shipToAddress,
		Type:                    typeGroup,
		PriceGroups:             apiresource.NewList(priceGroups, apiresource.PageInfo{}),
		ParentAccount:           parentAccount,
		CreatedAt:               grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:               grpcutil.TimestampToTime(c.UpdatedAt),
	}
}

func customerAddressPresenter(a *pb.CustomerAddressProto) *apiresource.Address {
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
		IsDropShip:  a.IsDropShip,
		Geolocation: geolocation,
		CreatedAt:   grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func CustomerSummaryPresenter(cs *pb.CustomerSummaryProto) apiresource.CustomerSummary {
	if cs == nil {
		return apiresource.CustomerSummary{}
	}

	return apiresource.CustomerSummary{
		ID:                cs.Id,
		Object:            constants.ObjectTypeCustomerSummary,
		Name:              cs.Name,
		Number:            cs.Number,
		Email:             cs.Email,
		CustomerTypeGroup: cs.CustomerTypeGroup,
		Status:            constants.AccountStatusCode(cs.Status),
		CreatedAt:         grpcutil.TimestampToTime(cs.CreatedAt),
	}
}

func CustomerListPresenter(resp *pb.ListCustomersResponse) *apiresource.List[apiresource.CustomerSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.CustomerSummary](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.CustomerSummary, len(resp.Customers))
	for i, cs := range resp.Customers {
		items[i] = CustomerSummaryPresenter(cs)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
