package apirequest

import (
	"github.com/open-mrp/apikit/field"

	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// An amount of money.
type MoneyInput struct {
	// Decimal amount, as a string to keep precision.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// Three-letter ISO 4217 currency code, lowercase. Only `usd` is supported; any other code fails with `currency_not_supported`.
	Currency string `json:"currency" validate:"required"`
}

// An amount of money per one unit, such as a price per pair or a labor rate per hour.
type UnitPriceInput struct {
	// Decimal amount, as a string to keep precision.
	Amount string `json:"amount" validate:"required" format:"decimal"`
	// Three-letter ISO 4217 currency code, lowercase. Only `usd` is supported; any other code fails with `currency_not_supported`.
	Currency string `json:"currency" validate:"required"`
	// ID of the unit the amount is charged per.
	PerUnitID string `json:"per_unit_id" validate:"required"`
}

// Something taken off a price or a total.
type DiscountInput struct {
	// How the discount is expressed.
	//
	// `percent_off` requires `percent_off`; `amount_off` requires `amount_off`.
	Type constants.DiscountType `json:"type" validate:"required"`
	// Whole-number percent to take off, from `0` to `100` (`"5"` is 5%).
	PercentOff field.Optional[string] `json:"percent_off,omitzero" format:"decimal"`
	// Fixed amount to take off.
	AmountOff field.Optional[MoneyInput] `json:"amount_off,omitzero"`
}

// ToProto converts the amount for core-service.
func (m MoneyInput) ToProto() *pb.Money {
	return &pb.Money{Amount: m.Amount, Currency: m.Currency}
}

// ToProto converts the price for core-service.
func (p UnitPriceInput) ToProto() *pb.UnitPrice {
	return &pb.UnitPrice{Amount: p.Amount, Currency: p.Currency, PerUnitId: p.PerUnitID}
}

// ToProto converts the rate for core-service.
func (r RateInput) ToProto() *pb.RateInput {
	return &pb.RateInput{Value: r.Value, UnitId: r.UnitID, PerUnitId: r.PerUnitID}
}

// ToProto converts the discount for core-service.
func (d DiscountInput) ToProto() *pb.Discount {
	out := &pb.Discount{Type: string(d.Type)}
	if v, ok := d.PercentOff.Value(); ok {
		out.PercentOff = &v
	}
	if v, ok := d.AmountOff.Value(); ok {
		out.AmountOff = v.ToProto()
	}
	return out
}

// OptionalMoneyToProto converts an optional amount, or returns nil when it is unset.
func OptionalMoneyToProto(f field.Optional[MoneyInput]) *pb.Money {
	v, ok := f.Value()
	if !ok {
		return nil
	}
	return v.ToProto()
}

// OptionalUnitPriceToProto converts an optional price, or returns nil when it is unset.
func OptionalUnitPriceToProto(f field.Optional[UnitPriceInput]) *pb.UnitPrice {
	v, ok := f.Value()
	if !ok {
		return nil
	}
	return v.ToProto()
}

// MoneyFieldToProto converts a clearable amount into its patch: nil when omitted, a clear instruction for an explicit null, otherwise the new amount.
func MoneyFieldToProto(f field.Clearable[MoneyInput]) *pb.MoneyPatch {
	if f.IsUnset() {
		return nil
	}
	if f.IsClear() {
		return &pb.MoneyPatch{Clear: true}
	}
	v, _ := f.Value()
	return &pb.MoneyPatch{Value: v.ToProto()}
}
