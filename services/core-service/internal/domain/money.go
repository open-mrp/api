package domain

import (
	"fmt"

	"github.com/open-mrp/apikit/apierror"
	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// Money is an amount in a currency.
type Money struct {
	Amount   string
	Currency constants.Currency
}

// UnitPrice is an amount of money per one unit.
type UnitPrice struct {
	Amount    string
	Currency  constants.Currency
	PerUnitID string
}

// Discount is something taken off: a whole-number percent or a fixed amount.
type Discount struct {
	Type       constants.DiscountType
	PercentOff *string
	AmountOff  *Money
}

// ToProto converts the amount for the gateway.
func (m Money) ToProto() *pb.Money {
	return &pb.Money{Amount: m.Amount, Currency: string(m.Currency)}
}

// ToProto converts the price for the gateway.
func (p UnitPrice) ToProto() *pb.UnitPrice {
	return &pb.UnitPrice{Amount: p.Amount, Currency: string(p.Currency), PerUnitId: p.PerUnitID}
}

// ToProto converts the discount for the gateway.
func (d Discount) ToProto() *pb.Discount {
	out := &pb.Discount{Type: string(d.Type), PercentOff: d.PercentOff}
	if d.AmountOff != nil {
		out.AmountOff = d.AmountOff.ToProto()
	}
	return out
}

// ValidateCurrency accepts the currencies the API supports (usd only) and refuses any other code with currency_not_supported on param.
func ValidateCurrency(param, currency string) (constants.Currency, *apierror.APIError) {
	c := constants.Currency(currency)
	if !c.IsValid() {
		return "", apperr.NewCurrencyNotSupportedError(param, fmt.Sprintf("Currency %q is not supported; use %q.", currency, constants.CurrencyUSD))
	}
	return c, nil
}

// MoneyFromProto validates a requested amount: a decimal amount in a supported currency.
func MoneyFromProto(param string, m *pb.Money) (*Money, *apierror.APIError) {
	if m == nil {
		return nil, nil
	}
	currency, apiErr := ValidateCurrency(param+".currency", m.Currency)
	if apiErr != nil {
		return nil, apiErr
	}
	if _, err := decimal.NewFromString(m.Amount); err != nil {
		return nil, apierror.NewInvalidFormatError(param+".amount", "Must be a decimal string.")
	}
	return &Money{Amount: m.Amount, Currency: currency}, nil
}

// UnitPriceFromProto validates a requested price: a decimal amount in a supported currency, per a unit.
func UnitPriceFromProto(param string, p *pb.UnitPrice) (*UnitPrice, *apierror.APIError) {
	if p == nil {
		return nil, nil
	}
	money, apiErr := MoneyFromProto(param, &pb.Money{Amount: p.Amount, Currency: p.Currency})
	if apiErr != nil {
		return nil, apiErr
	}
	return &UnitPrice{Amount: money.Amount, Currency: money.Currency, PerUnitID: p.PerUnitId}, nil
}

// DiscountFromProto validates a requested discount: percent_off between 0 and 100, or a fixed amount_off.
func DiscountFromProto(param string, d *pb.Discount) (*Discount, *apierror.APIError) {
	if d == nil {
		return nil, nil
	}
	switch constants.DiscountType(d.Type) {
	case constants.DiscountTypePercentOff:
		if d.PercentOff == nil || d.AmountOff != nil {
			return nil, apierror.NewInvalidFormatError(param+".percent_off", "A percent_off discount takes percent_off and no amount_off.")
		}
		pct, err := decimal.NewFromString(*d.PercentOff)
		if err != nil || pct.IsNegative() || pct.GreaterThan(decimal.NewFromInt(100)) {
			return nil, apierror.NewInvalidFormatError(param+".percent_off", "Must be a decimal from 0 to 100.")
		}
		return &Discount{Type: constants.DiscountTypePercentOff, PercentOff: d.PercentOff}, nil
	case constants.DiscountTypeAmountOff:
		if d.AmountOff == nil || d.PercentOff != nil {
			return nil, apierror.NewInvalidFormatError(param+".amount_off", "An amount_off discount takes amount_off and no percent_off.")
		}
		amount, apiErr := MoneyFromProto(param+".amount_off", d.AmountOff)
		if apiErr != nil {
			return nil, apiErr
		}
		return &Discount{Type: constants.DiscountTypeAmountOff, AmountOff: amount}, nil
	default:
		return nil, apierror.NewInvalidFormatError(param+".type", "Must be percent_off or amount_off.")
	}
}
