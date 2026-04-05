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
		recipientAccount = &apiresource.Customer{
			ID:     ap.RecipientAccount.Id,
			Object: constants.ObjectTypeCustomer,
			Name:   ap.RecipientAccount.Name,
		}
	}

	var productLine *apiresource.ProductLine
	if ap.ProductLine != nil {
		productLine = &apiresource.ProductLine{
			ID:     ap.ProductLine.Id,
			Object: constants.ObjectTypeProductLine,
			Name:   ap.ProductLine.Name,
		}
	}

	var rate *apiresource.Rate
	if ap.Rate != nil {
		rate = &apiresource.Rate{
			ID:     ap.Rate.Id,
			Object: constants.ObjectTypeRate,
			Value:  ap.Rate.Value,
		}
		var numeratorAbbreviation, numeratorUnitType, denominatorAbbreviation string
		if ap.Rate.NumeratorUnit != nil {
			rate.NumeratorUnit = &apiresource.Unit{
				ID:           ap.Rate.NumeratorUnit.Id,
				Object:       constants.ObjectTypeUnit,
				Name:         ap.Rate.NumeratorUnit.Name,
				Abbreviation: ap.Rate.NumeratorUnit.Abbreviation,
				Type:         constants.UnitType(ap.Rate.NumeratorUnit.Type),
			}
			numeratorAbbreviation = ap.Rate.NumeratorUnit.Abbreviation
			numeratorUnitType = ap.Rate.NumeratorUnit.Type
		}
		if ap.Rate.DenominatorUnit != nil {
			rate.DenominatorUnit = &apiresource.Unit{
				ID:           ap.Rate.DenominatorUnit.Id,
				Object:       constants.ObjectTypeUnit,
				Name:         ap.Rate.DenominatorUnit.Name,
				Abbreviation: ap.Rate.DenominatorUnit.Abbreviation,
				Type:         constants.UnitType(ap.Rate.DenominatorUnit.Type),
			}
			denominatorAbbreviation = ap.Rate.DenominatorUnit.Abbreviation
		}
		rate.DisplayValue = apiresource.FormatRateDisplayValue(ap.Rate.Value, numeratorAbbreviation, numeratorUnitType, denominatorAbbreviation)
	}

	categories := make([]apiresource.ItemCategory, len(ap.Categories))
	for i, c := range ap.Categories {
		categories[i] = apiresource.ItemCategory{
			ID:     c.Id,
			Object: constants.ObjectTypeItemCategory,
			Name:   c.Name,
		}
	}

	attributes := make([]apiresource.Attribute, len(ap.Attributes))
	for i, a := range ap.Attributes {
		attributes[i] = apiresource.Attribute{
			ID:     a.Id,
			Object: constants.ObjectTypeAttribute,
			Value:  a.Value,
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
