package domain

import (
	"testing"

	"github.com/open-mrp/apikit/apierror"

	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func TestMoneyFromProto(t *testing.T) {
	t.Parallel()

	m, apiErr := MoneyFromProto("credit_limit", &pb.Money{Amount: "10.50", Currency: "usd"})
	if apiErr != nil || m.Amount != "10.50" || m.Currency != constants.CurrencyUSD {
		t.Fatalf("MoneyFromProto = %+v, %v", m, apiErr)
	}

	_, apiErr = MoneyFromProto("credit_limit", &pb.Money{Amount: "10.50", Currency: "eur"})
	if apiErr == nil || apiErr.Code != apperr.CodeCurrencyNotSupported || apiErr.Param != "credit_limit.currency" {
		t.Errorf("eur: got %v, want currency_not_supported on credit_limit.currency", apiErr)
	}

	_, apiErr = MoneyFromProto("credit_limit", &pb.Money{Amount: "ten", Currency: "usd"})
	if apiErr == nil || apiErr.Param != "credit_limit.amount" {
		t.Errorf("non-decimal amount: got %v, want an error on credit_limit.amount", apiErr)
	}

	if m, apiErr := MoneyFromProto("credit_limit", nil); m != nil || apiErr != nil {
		t.Errorf("nil money = %+v, %v; want nothing", m, apiErr)
	}
}

func TestUnitPriceRateParams(t *testing.T) {
	t.Parallel()

	p, apiErr := UnitPriceFromProto("unit_price", &pb.UnitPrice{Amount: "18.45", Currency: "usd", PerUnitId: "un_pr"})
	if apiErr != nil {
		t.Fatal(apiErr)
	}
	if got := p.RateParams("dollar"); got != (CreateRateParams{Value: "18.45", NumeratorUnitID: "dollar", DenominatorUnitID: "un_pr"}) {
		t.Errorf("RateParams = %+v", got)
	}
	if got := UnitPriceFromCurrencyRate("18.45", "un_pr"); got != (UnitPrice{Amount: "18.45", Currency: constants.CurrencyUSD, PerUnitID: "un_pr"}) {
		t.Errorf("UnitPriceFromCurrencyRate = %+v", got)
	}
}

func TestDiscountFromProto(t *testing.T) {
	t.Parallel()

	pct := "5"
	d, apiErr := DiscountFromProto("discount", &pb.Discount{Type: "percent_off", PercentOff: &pct})
	if apiErr != nil || d.Type != constants.DiscountTypePercentOff || *d.PercentOff != "5" {
		t.Fatalf("percent_off = %+v, %v", d, apiErr)
	}

	d, apiErr = DiscountFromProto("discount", &pb.Discount{Type: "amount_off", AmountOff: &pb.Money{Amount: "2.00", Currency: "usd"}})
	if apiErr != nil || d.AmountOff == nil || d.AmountOff.Amount != "2.00" {
		t.Fatalf("amount_off = %+v, %v", d, apiErr)
	}

	over := "101"
	for name, in := range map[string]*pb.Discount{
		"over 100":       {Type: "percent_off", PercentOff: &over},
		"missing amount": {Type: "amount_off"},
		"both":           {Type: "percent_off", PercentOff: &pct, AmountOff: &pb.Money{Amount: "1", Currency: "usd"}},
		"unknown type":   {Type: "free"},
	} {
		if _, apiErr := DiscountFromProto("discount", in); apiErr == nil || apiErr.Code != apierror.CodeValidationFailed {
			t.Errorf("%s: got %v, want validation_failed", name, apiErr)
		}
	}
}
