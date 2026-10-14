package grpc

import (
	"context"

	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/field"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/fieldpb"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// moneyRows converts requested money into the quantity and rate rows money is still stored as, in the currency's
// unit. It keeps the first error, so a handler converts a whole request and then checks err once. Temporary, with
// domain/money_adapter.go.
type moneyRows struct {
	ctx            context.Context
	units          domain.UnitSvc
	currencyUnitID string
	apiErr         *apierror.APIError
}

// newMoneyRows converts money for one request, with the unit service registered on the core handler.
func newMoneyRows(ctx context.Context) *moneyRows {
	return &moneyRows{ctx: ctx, units: handler.unitSvc}
}

func (c *moneyRows) unitID(currency constants.Currency) string {
	if c.currencyUnitID == "" && c.apiErr == nil {
		c.currencyUnitID, c.apiErr = c.units.CurrencyUnitID(c.ctx, currency)
	}
	return c.currencyUnitID
}

// rate returns a requested price as a rate from the currency's unit, or nil when none was sent.
func (c *moneyRows) rate(param string, p *pb.UnitPrice) *domain.CreateRateParams {
	if p == nil || c.apiErr != nil {
		return nil
	}
	price, apiErr := domain.UnitPriceFromProto(param, p)
	if apiErr != nil {
		c.apiErr = apiErr
		return nil
	}
	unitID := c.unitID(price.Currency)
	if c.apiErr != nil {
		return nil
	}
	rate := price.RateParams(unitID)
	return &rate
}

// quantity returns a requested amount as a quantity in the currency's unit, or nil when none was sent.
func (c *moneyRows) quantity(param string, m *pb.Money) *domain.QuantityInput {
	if m == nil || c.apiErr != nil {
		return nil
	}
	money, apiErr := domain.MoneyFromProto(param, m)
	if apiErr != nil {
		c.apiErr = apiErr
		return nil
	}
	unitID := c.unitID(money.Currency)
	if c.apiErr != nil {
		return nil
	}
	return &domain.QuantityInput{Value: money.Amount, UnitID: unitID}
}

// patch returns a requested change to an amount: unset when omitted, clear when cleared, otherwise the new amount as a
// quantity in the currency's unit.
func (c *moneyRows) patch(param string, p *pb.MoneyPatch) field.Clearable[domain.QuantityInput] {
	if p == nil {
		return field.Unset[domain.QuantityInput]()
	}
	if p.Clear || p.Value == nil {
		return field.Clear[domain.QuantityInput]()
	}
	q := c.quantity(param, p.Value)
	if q == nil {
		return field.Unset[domain.QuantityInput]()
	}
	return field.Set(*q)
}

// fieldPatch is patch for services that take a fieldpb.QuantityInput.
func (c *moneyRows) fieldPatch(param string, p *pb.MoneyPatch) field.Clearable[fieldpb.QuantityInput] {
	f := c.patch(param, p)
	if f.IsUnset() {
		return field.Unset[fieldpb.QuantityInput]()
	}
	q, ok := f.Value()
	if !ok {
		return field.Clear[fieldpb.QuantityInput]()
	}
	return field.Set(fieldpb.QuantityInput{Value: q.Value, UnitID: q.UnitID})
}

func (c *moneyRows) err() *apierror.APIError {
	return c.apiErr
}

// The rate* helpers read a converted price into domain params that still carry a rate's three parts, as values for a
// required price and as pointers (nil when no price was sent) for an optional one.

func rateValue(r *domain.CreateRateParams) string {
	if r == nil {
		return ""
	}
	return r.Value
}

func rateUnit(r *domain.CreateRateParams) string {
	if r == nil {
		return ""
	}
	return r.NumeratorUnitID
}

func ratePerUnit(r *domain.CreateRateParams) string {
	if r == nil {
		return ""
	}
	return r.DenominatorUnitID
}

func rateValuePtr(r *domain.CreateRateParams) *string {
	if r == nil {
		return nil
	}
	return &r.Value
}

func rateUnitPtr(r *domain.CreateRateParams) *string {
	if r == nil {
		return nil
	}
	return &r.NumeratorUnitID
}

func ratePerUnitPtr(r *domain.CreateRateParams) *string {
	if r == nil {
		return nil
	}
	return &r.DenominatorUnitID
}

// unitPriceToProto presents a price stored as a currency rate.
func unitPriceToProto(value, denominatorUnitID string) *pb.UnitPrice {
	return domain.UnitPriceFromCurrencyRate(value, denominatorUnitID).ToProto()
}

// itemPriceToProto presents an item's unit value or unit cost, stored as a currency rate, as a price.
func itemPriceToProto(r *domain.Rate) *pb.UnitPrice {
	if r == nil {
		return nil
	}
	return unitPriceToProto(r.Value, r.DenominatorUnitID)
}

// moneyToProto presents an amount stored as a quantity in a currency unit.
func moneyToProto(value string) *pb.Money {
	return domain.MoneyFromCurrencyQuantity(value).ToProto()
}

// moneyQuantityToProto presents an amount stored as a quantity in a currency unit, or nil when there is none.
func moneyQuantityToProto(q *domain.Quantity) *pb.Money {
	if q == nil {
		return nil
	}
	return moneyToProto(q.Value)
}
