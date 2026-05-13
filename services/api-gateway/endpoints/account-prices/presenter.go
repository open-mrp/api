package accountpriceep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func AccountPricePresenter(ap *pb.AccountPriceInfo) apiresource.AccountPrice {
	if ap == nil {
		return apiresource.AccountPrice{}
	}

	var recipientAccount *apiresource.Customer
	if ap.RecipientAccount != nil {
		ra := ap.RecipientAccount
		ediStatus := constants.EDIStatusDisabled
		if ra.IsEdiEnabled {
			ediStatus = constants.EDIStatusEnabled
		}
		recipientAccount = &apiresource.Customer{
			ID:               ra.Id,
			Object:           constants.ObjectTypeCustomer,
			Name:             ra.Name,
			Number:           ra.Number,
			Status:           constants.AccountStatusCode(ra.Status),
			EDIStatus:        ediStatus,
			RelationshipType: constants.CustomerRelationshipType(ra.RelationshipType),
			CommissionPolicy: constants.CommissionPolicy(ra.CommissionPolicy),
			CreatedAt:        grpcutil.TimestampToTime(ra.CreatedAt),
			UpdatedAt:        grpcutil.TimestampToTime(ra.UpdatedAt),
		}
	}

	var productLine *apiresource.ProductLine
	if ap.ProductLine != nil {
		pl := ap.ProductLine
		productLine = &apiresource.ProductLine{
			ID:               pl.Id,
			Object:           constants.ObjectTypeProductLine,
			Name:             pl.Name,
			CommissionPolicy: constants.CommissionPolicy(pl.CommissionPolicy),
			FreightPolicy:    constants.FreightPolicy(pl.FreightPolicy),
			CreatedAt:        grpcutil.TimestampToTime(pl.CreatedAt),
			UpdatedAt:        grpcutil.TimestampToTime(pl.UpdatedAt),
		}
	}

	var rate *apiresource.Rate
	if ap.Rate != nil {
		rate = &apiresource.Rate{
			ID:        ap.Rate.Id,
			Object:    constants.ObjectTypeRate,
			Value:     ap.Rate.Value,
			CreatedAt: grpcutil.TimestampToTime(ap.Rate.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(ap.Rate.UpdatedAt),
		}
		var numeratorAbbreviation, numeratorUnitType, denominatorAbbreviation string
		if ap.Rate.NumeratorUnit != nil {
			nu := ap.Rate.NumeratorUnit
			rate.NumeratorUnit = &apiresource.Unit{
				ID:                nu.Id,
				Object:            constants.ObjectTypeUnit,
				Name:              nu.Name,
				Abbreviation:      nu.Abbreviation,
				Type:              constants.UnitType(nu.Type),
				RatioNumerator:    nu.RatioNumerator,
				RatioDenominator:  nu.RatioDenominator,
				OffsetNumerator:   nu.OffsetNumerator,
				OffsetDenominator: nu.OffsetDenominator,
				CreatedAt:         grpcutil.TimestampToTime(nu.CreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(nu.UpdatedAt),
			}
			numeratorAbbreviation = nu.Abbreviation
			numeratorUnitType = nu.Type
		}
		if ap.Rate.DenominatorUnit != nil {
			du := ap.Rate.DenominatorUnit
			rate.DenominatorUnit = &apiresource.Unit{
				ID:                du.Id,
				Object:            constants.ObjectTypeUnit,
				Name:              du.Name,
				Abbreviation:      du.Abbreviation,
				Type:              constants.UnitType(du.Type),
				RatioNumerator:    du.RatioNumerator,
				RatioDenominator:  du.RatioDenominator,
				OffsetNumerator:   du.OffsetNumerator,
				OffsetDenominator: du.OffsetDenominator,
				CreatedAt:         grpcutil.TimestampToTime(du.CreatedAt),
				UpdatedAt:         grpcutil.TimestampToTime(du.UpdatedAt),
			}
			denominatorAbbreviation = du.Abbreviation
		}
		rate.DisplayValue = apiresource.FormatRateDisplayValue(ap.Rate.Value, numeratorAbbreviation, numeratorUnitType, denominatorAbbreviation)
	}

	categories := make([]apiresource.ItemCategory, len(ap.Categories))
	for i, c := range ap.Categories {
		categories[i] = apiresource.ItemCategory{
			ID:        c.Id,
			Object:    constants.ObjectTypeItemCategory,
			Name:      c.Name,
			Type:      constants.ItemCategoryType(c.Type),
			CreatedAt: grpcutil.TimestampToTime(c.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(c.UpdatedAt),
		}
	}

	attributes := make([]apiresource.Attribute, len(ap.Attributes))
	for i, a := range ap.Attributes {
		attributes[i] = apiresource.Attribute{
			ID:        a.Id,
			Object:    constants.ObjectTypeAttribute,
			Value:     a.Value,
			ColorCode: constants.Color(a.ColorCode),
			CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
		}
	}

	return apiresource.AccountPrice{
		ID:               ap.Id,
		Object:           constants.ObjectTypeAccountPrice,
		RecipientAccount: recipientAccount,
		ProductLine:      productLine,
		Rate:             rate,
		Categories:       apiresource.NewList(categories, apiresource.PageInfo{}),
		Attributes:       apiresource.NewList(attributes, apiresource.PageInfo{}),
		CreatedAt:        grpcutil.TimestampToTime(ap.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(ap.UpdatedAt),
	}
}

func AccountPriceListPresenter(resp *pb.ListAccountPricesResponse) *apiresource.List[apiresource.AccountPrice] {
	if resp == nil {
		return apiresource.NewList[apiresource.AccountPrice](nil, apiresource.PageInfo{})
	}

	prices := make([]apiresource.AccountPrice, len(resp.AccountPrices))
	for i, ap := range resp.AccountPrices {
		prices[i] = AccountPricePresenter(ap)
	}

	return apiresource.NewList(prices, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
