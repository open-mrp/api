package resourceloaders

import (
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// MoneyFromProto presents an amount from core-service with its amount normalized to cents, or nil when absent.
func MoneyFromProto(m *pb.Money) *apiresource.Money {
	if m == nil {
		return nil
	}
	return &apiresource.Money{Amount: apiresource.NormalizeMonetaryQuantityValue(m.Amount), Currency: constants.Currency(m.Currency)}
}

// ExactMoneyFromProto presents an amount from core-service keeping every digit, or nil when absent.
func ExactMoneyFromProto(m *pb.Money) *apiresource.Money {
	if m == nil {
		return nil
	}
	return &apiresource.Money{Amount: m.Amount, Currency: constants.Currency(m.Currency)}
}

// UnitPriceFromProto presents a price from core-service, or nil when absent. Its per-unit loads before the response is written.
func UnitPriceFromProto(p *pb.UnitPrice) *apiresource.UnitPrice {
	if p == nil {
		return nil
	}
	return &apiresource.UnitPrice{
		Amount:   apiresource.NormalizeRateValue(p.Amount),
		Currency: constants.Currency(p.Currency),
		PerUnit:  apiresource.UnitStub(p.PerUnitId),
	}
}

// ExactUnitPriceFromProto presents a computed price keeping every digit, so it equals what an order will charge.
func ExactUnitPriceFromProto(p *pb.UnitPrice) *apiresource.UnitPrice {
	if p == nil {
		return nil
	}
	return apiresource.NewUnitPrice(p.Amount, p.PerUnitId)
}

// RateFromProto presents a stored rate, or nil when absent. Its units load before the response is written.
func RateFromProto(r *pb.RateInfo) *apiresource.Rate {
	if r == nil {
		return nil
	}
	return apiresource.NewRate(apiresource.NormalizeRateValue(r.Value), r.NumeratorUnitId, r.DenominatorUnitId)
}

// QuantityFromProto presents a stored quantity, or nil when absent. Its unit loads before the response is written.
func QuantityFromProto(q *pb.QuantityInfo) *apiresource.Quantity {
	if q == nil {
		return nil
	}
	return apiresource.NewQuantity(apiresource.NormalizeQuantityValue(q.Value, q.UnitType), q.UnitId)
}
