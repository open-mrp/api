package customerep

import (
	"context"

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
			d.Priority = &apiresource.Priority{
				ID:     c.DefaultPriority.Id,
				Code:   constants.PriorityCode(c.DefaultPriority.Code),
				Object: constants.ObjectTypePriority,
				Name:   c.DefaultPriority.Name,
			}
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

		defaults = d
	}

	// Credit limit
	var creditLimit *apiresource.Quantity
	if c.CreditLimit != nil {
		unitType := c.CreditLimit.UnitType
		creditLimit = &apiresource.Quantity{
			ID:           c.CreditLimit.Id,
			Object:       constants.ObjectTypeQuantity,
			Value:        apiresource.NormalizeQuantityValue(c.CreditLimit.Value, unitType),
			DisplayValue: apiresource.FormatDisplayValue(c.CreditLimit.Value, c.CreditLimit.UnitAbbreviation, unitType),
			Unit: &apiresource.Unit{
				ID:           c.CreditLimit.UnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         c.CreditLimit.UnitName,
				Abbreviation: c.CreditLimit.UnitAbbreviation,
				Type:         constants.UnitType(unitType),
			},
		}
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
		tg := &apiresource.AccountGroup{
			ID:               c.TypeGroup.Id,
			Object:           constants.ObjectTypeAccountGroup,
			Name:             c.TypeGroup.Name,
			CommissionPolicy: constants.CommissionPolicy(c.TypeGroup.CommissionPolicy),
			FreightPolicy:    constants.FreightPolicy(c.TypeGroup.FreightPolicy),
			Type:             constants.AccountGroupType(c.TypeGroup.Type),
		}
		if c.TypeGroup.CreatedAt != nil {
			tg.CreatedAt = c.TypeGroup.CreatedAt.AsTime()
		}
		if c.TypeGroup.UpdatedAt != nil {
			tg.UpdatedAt = c.TypeGroup.UpdatedAt.AsTime()
		}
		typeGroup = tg
	}

	// Price groups
	var priceGroups []apiresource.AccountGroup
	for _, pg := range c.PriceGroups {
		item := apiresource.AccountGroup{
			ID:               pg.Id,
			Object:           constants.ObjectTypeAccountGroup,
			Name:             pg.Name,
			CommissionPolicy: constants.CommissionPolicy(pg.CommissionPolicy),
			FreightPolicy:    constants.FreightPolicy(pg.FreightPolicy),
			Type:             constants.AccountGroupType(pg.Type),
		}
		if pg.CreatedAt != nil {
			item.CreatedAt = pg.CreatedAt.AsTime()
		}
		if pg.UpdatedAt != nil {
			item.UpdatedAt = pg.UpdatedAt.AsTime()
		}
		priceGroups = append(priceGroups, item)
	}

	// Parent account
	var parentAccount *apiresource.Customer
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
		parentAccount = pa
	}

	// Child accounts
	var childAccounts *apiresource.List[apiresource.Customer]
	if len(c.ChildAccounts) > 0 {
		items := make([]apiresource.Customer, len(c.ChildAccounts))
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
			items[i] = ca
		}
		childAccounts = apiresource.NewList(items, apiresource.PageInfo{})
	}

	return apiresource.Customer{
		ID:                      c.Id,
		Object:                  constants.ObjectTypeCustomer,
		Name:                    c.Name,
		Number:                  c.Number,
		Status:                  constants.AccountStatusCode(c.Status),
		EDIStatus:               ediStatusFromBool(c.IsEdiEnabled),
		RelationshipType:        customerRelationshipType(c.IsParentAccount, parentAccount),
		CommissionPolicy:        constants.CommissionPolicy(c.CommissionPolicy),
		Note:                    c.Note,
		CreditLimit:             creditLimit,
		ContactInfo:             contactInfo,
		FreightPreferences:      freightPreferences,
		Defaults:                defaults,
		NotificationPreferences: notificationPreferences,
		BillToAddress:           billToAddress,
		ShipToAddress:           shipToAddress,
		Type:                    typeGroup,
		PriceGroups:             apiresource.NewList(priceGroups, apiresource.PageInfo{}),
		ParentAccount:           parentAccount,
		ChildAccounts:           childAccounts,
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
		Type:        addressTypeFromDropShip(a.IsDropShip),
		Geolocation: geolocation,
		CreatedAt:   grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func ediStatusFromBool(enabled bool) constants.EDIStatus {
	if enabled {
		return constants.EDIStatusEnabled
	}
	return constants.EDIStatusDisabled
}

func customerRelationshipType(isParent bool, parentAccount *apiresource.Customer) constants.CustomerRelationshipType {
	if isParent {
		return constants.CustomerRelationshipTypeParent
	}
	if parentAccount != nil {
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

func CustomerListPresenter(ctx context.Context, resp *pb.ListCustomersResponse) *apiresource.List[apiresource.Customer] {
	if resp == nil {
		return apiresource.NewList[apiresource.Customer](nil, apiresource.PageInfo{})
	}

	items := make([]apiresource.Customer, len(resp.Customers))
	for i, c := range resp.Customers {
		items[i] = CustomerPresenter(c)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
