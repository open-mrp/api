package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/pricing"
)

// The invoice a customer receives must bill the amount the dashboard shows. A line priced per pair
// but invoiced in cartons of twelve printed $67.50 where the dashboard showed $810.00, because the
// carton count was multiplied by the per-pair price without converting it.

// cartonInvoice is the shipped invoice that exposed the bug: three cartons of twelve pairs at $22.50
// a pair, plus a shipping line priced per its own unit.
func cartonInvoice() (*domain.Invoice, []*domain.InvoiceLine) {
	invoice := &domain.Invoice{
		ID:         "inv_1",
		Number:     "23960",
		CustomerID: "acct_customer",
		CreatedAt:  time.Date(2026, 9, 8, 17, 39, 0, 0, time.UTC),
	}
	lines := []*domain.InvoiceLine{
		{
			ID:                   "il_1",
			OrderLineItemNumber:  poPtr(int32(1)),
			OrderLineItemSKU:     poPtr("LTD3612SN"),
			OrderLineDescription: poPtr("20-30 mmHg, Full Length Thigh, Open Toe, Silky Nude, Size 2"),
			QuantityValue:        "3",
			QuantityUnitID:       "unit_carton",
			QuantityUnitName:     "Carton (12 pr)",
			OrderLineQtyOrdered:  "3",
			UnitPriceValue:       "22.5",
			UnitPriceDenUnit:     "unit_pair",
			UnitPriceDenUnitAbbr: "pr",
			// A carton is twelve pairs.
			PricingQuantityRatioNumerator: "12", PricingQuantityRatioDenominator: "1", PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1",
		},
		{
			ID:                            "il_2",
			OrderLineItemNumber:           poPtr(int32(2)),
			OrderLineItemSKU:              poPtr("Shipping"),
			OrderLineDescription:          poPtr("Shipping charges"),
			QuantityValue:                 "1",
			QuantityUnitID:                "unit_each",
			QuantityUnitName:              "Each",
			OrderLineQtyOrdered:           "1",
			UnitPriceValue:                "0",
			UnitPriceDenUnit:              "unit_each",
			UnitPriceDenUnitAbbr:          "ea",
			PricingQuantityRatioNumerator: "1", PricingQuantityRatioDenominator: "1", PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1",
		},
	}
	return invoice, lines
}

func TestBuildInvoiceDoc_PricesInTheRateUnit(t *testing.T) {
	t.Parallel()

	invoice, lines := cartonInvoice()
	doc := buildInvoiceDoc(invoice, lines, nil, nil, nil, nil, nil, invoiceDocLookups{
		Conversions: map[string]pricing.UnitConversion{"il_1": twelveToOne},
	})

	require.Len(t, doc.Lines, 2)
	require.Equal(t, "$810.00", doc.Lines[0].Total, "three cartons of twelve at $22.50 a pair")
	// The quantity columns still read in the unit the customer was invoiced in.
	require.Equal(t, "3", doc.Lines[0].Invoiced)
	require.Equal(t, "Carton (12 pr)", doc.Lines[0].Unit)
	require.Equal(t, "$0.00", doc.Lines[1].Total)
	require.Equal(t, "$810.00", doc.OrderTotal)
}

func TestGatherInvoiceDoc_ConvertsQuantitiesAndReadsTheCustomerPhone(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repos := invoiceDocRepos(ctrl)
	repos.customers.EXPECT().Get(gomock.Any(), "acct_seller", "acct_customer", gomock.Nil()).
		Return(&domain.Customer{Phone: poPtr("832-677-6212")}, nil)

	invoice, lines := cartonInvoice()
	doc, apiErr := gatherInvoiceDoc(context.Background(), repos.factory, "acct_seller", invoice, lines)

	require.Nil(t, apiErr)
	require.Equal(t, "$810.00", doc.OrderTotal)
	require.Equal(t, "832-677-6212", doc.Header.ContactPhone)
	// The seller is in New York, so the 17:39 UTC ship reads as the afternoon it was.
	require.Equal(t, "09/08/2026 01:39 PM", doc.Header.OrderDateLong)
}

// A line that cannot be priced must stop the document: the alternative is mailing the wrong amount.
func TestGatherInvoiceDoc_FailsWhenALineCannotBePriced(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repos := invoiceDocRepos(ctrl)

	invoice, lines := cartonInvoice()
	// A unit with a zero ratio cannot be converted to or from.
	lines[0].PricingQuantityRatioDenominator = "0"
	_, apiErr := gatherInvoiceDoc(context.Background(), repos.factory, "acct_seller", invoice, lines)

	require.NotNil(t, apiErr)
}

func TestInDocumentZone(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 9, 8, 17, 39, 0, 0, time.UTC)

	t.Run("reads the time in the origin's zone", func(t *testing.T) {
		got := inDocumentZone(at, &domain.ShippingAddress{Timezone: poPtr("America/New_York")})
		require.Equal(t, "09/08/2026 01:39 PM", got.Format("01/02/2006 03:04 PM"))
	})

	t.Run("falls back to the server zone without one", func(t *testing.T) {
		require.Equal(t, at.Local(), inDocumentZone(at, nil))
		require.Equal(t, at.Local(), inDocumentZone(at, &domain.ShippingAddress{}))
		require.Equal(t, at.Local(), inDocumentZone(at, &domain.ShippingAddress{Timezone: poPtr("Not/AZone")}))
	})
}

// The bill-to block lists the contacts as the dashboard does: emails lowercased, then the customer's
// phone. The text is set in a UTF-8 font, so a truncated value carries a real ellipsis.
func TestInvoicePDFBillToContactsAndEncoding(t *testing.T) {
	t.Parallel()

	invoice, lines, order := invoiceFixture()
	lines[0].OrderLineItemSKU = poPtr("SOCK-CREW-BLACK-LARGE-EXTENDED-CALF-0000000000000000000000000000000000001")
	doc := buildInvoiceDoc(invoice, lines, order, &domain.Account{Name: "Acme Company"}, nil, nil,
		[]string{"AP@TexasEva.com"}, invoiceDocLookups{CustomerPhone: "832-677-6212 (AP & Purchasing)"})

	out, err := buildInvoicePDF(doc)
	require.NoError(t, err)

	runs := pdfText(t, out)
	require.True(t, pdfContains(runs, "ap@texaseva.com"), "emails are lowercased\n%s", pdfJoined(runs))
	require.True(t, pdfContains(runs, "832-677-6212 (AP & Purchasing)"), "the customer phone follows the emails\n%s", pdfJoined(runs))
	require.Contains(t, pdfJoined(runs), "…", "an overlong SKU is cut with an ellipsis")
	require.NotContains(t, pdfJoined(runs), "â€", "no cp1252 mojibake")
}

type invoiceDocRepoMocks struct {
	factory   *factorymock.MockRepoFactory
	customers *repositorymock.MockCustomerRepo
}

// invoiceDocRepos stubs every lookup gatherInvoiceDoc makes, leaving the customer for the test to
// set.
func invoiceDocRepos(ctrl *gomock.Controller) invoiceDocRepoMocks {
	m := invoiceDocRepoMocks{
		factory:   factorymock.NewMockRepoFactory(ctrl),
		customers: repositorymock.NewMockCustomerRepo(ctrl),
	}

	accounts := repositorymock.NewMockAccountRepo(ctrl)
	accounts.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&domain.Account{Name: "Acme Company"}, nil).AnyTimes()
	orders := repositorymock.NewMockSalesOrderRepo(ctrl)
	orders.EXPECT().GetAccountOriginAddress(gomock.Any(), gomock.Any()).
		Return(&domain.ShippingAddress{Street1: "601 Forum Parkway", City: "Rural Hall", State: "NC", Zip: "27045", Timezone: poPtr("America/New_York")}, nil).AnyTimes()
	invoices := repositorymock.NewMockInvoiceRepo(ctrl)
	invoices.EXPECT().GetEmailRecipients(gomock.Any(), gomock.Any()).Return([]string{"jamie@texaseva.com"}, nil).AnyTimes()

	m.factory.EXPECT().NewAccountRepo().Return(accounts).AnyTimes()
	m.factory.EXPECT().NewSalesOrderRepo().Return(orders).AnyTimes()
	m.factory.EXPECT().NewInvoiceRepo().Return(invoices).AnyTimes()
	m.factory.EXPECT().NewCustomerRepo().Return(m.customers).AnyTimes()
	return m
}

// The acknowledgement and the purchase order total their lines the same way: in the price's unit.
func TestRecordDocsPriceInTheRateUnit(t *testing.T) {
	t.Parallel()

	t.Run("order acknowledgement", func(t *testing.T) {
		order := &domain.SalesOrder{Number: "1001"}
		lines := []*domain.SalesOrderLine{
			{ID: "sol_1", LineItemNumber: 1, QuantityValue: "3", QuantityUnitID: "unit_carton", QuantityUnitName: "Carton (12 pr)",
				UnitPriceValue: "22.5", UnitPriceDenominatorUnitID: "unit_pair", UnitPriceDenominatorUnitAbbr: "pr"},
			{ID: "sol_2", LineItemNumber: 2, QuantityValue: "2", QuantityUnitID: "unit_each", QuantityUnitName: "each",
				UnitPriceValue: "5", UnitPriceDenominatorUnitID: "unit_each", UnitPriceDenominatorUnitAbbr: "ea"},
		}
		data := buildOrderAcknowledgementData(order, lines, map[string]pricing.UnitConversion{"sol_1": twelveToOne}, nil, nil)

		require.Equal(t, "$810.00", data.Lines[0].Total)
		require.Equal(t, "3 Carton (12 pr)", data.Lines[0].Qty, "the quantity still reads in the ordered unit")
		require.Equal(t, "$10.00", data.Lines[1].Total)
		require.Equal(t, "$820.00", data.OrderTotal)
	})

	t.Run("purchase order", func(t *testing.T) {
		order := &domain.PurchaseOrder{Number: "417"}
		lines := []*domain.PurchaseOrderLine{
			{ID: "pol_1", LineItemNumber: 1, QuantityValue: "1200", QuantityUnitID: "unit_pair", QuantityUnitName: "pair",
				UnitPriceValue: "8.5", UnitPriceDenominatorUnitID: "unit_dozen", UnitPriceDenominatorUnitAbbr: "dz"},
		}
		doc := buildPurchaseOrderDoc(order, lines, map[string]pricing.UnitConversion{"pol_1": pricing.Between(decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.NewFromInt(12), decimal.NewFromInt(1))}, nil, nil, nil)

		require.Equal(t, "$850.00", doc.Header.Lines[0].Total, "1,200 pairs at $8.50 a dozen")
		require.Equal(t, "$850.00", doc.Header.OrderTotal)
	})
}

var twelveToOne = pricing.Between(decimal.NewFromInt(12), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.NewFromInt(1))

func TestSalesOrderLineConversions(t *testing.T) {
	t.Parallel()

	convs, apiErr := salesOrderLineConversions([]*domain.SalesOrderLine{
		{ID: "sol_1", PricingQuantityRatioNumerator: "12", PricingQuantityRatioDenominator: "1", PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1"},
	})
	require.Nil(t, apiErr)
	require.Equal(t, "36", pricing.ExtendedPrice(decimal.NewFromInt(3), decimal.NewFromInt(1), convs["sol_1"]).String())

	_, apiErr = salesOrderLineConversions([]*domain.SalesOrderLine{
		{ID: "sol_1", PricingQuantityRatioNumerator: "12", PricingQuantityRatioDenominator: "0", PricingPriceRatioNumerator: "1", PricingPriceRatioDenominator: "1"},
	})
	require.NotNil(t, apiErr, "a line that cannot be priced fails the lot")

	require.Equal(t, pricing.Identity, conversionFor(nil, "missing"))
}

func TestUnitPairConversions(t *testing.T) {
	t.Parallel()

	carton := unitPair{Quantity: "unit_carton", PriceNumerator: "unit_usd", Price: "unit_pair"}
	same := unitPair{Quantity: "unit_each", PriceNumerator: "unit_usd", Price: "unit_each"}
	count := func(ratio int64, base bool) domain.UnitFactors {
		return domain.UnitFactors{RatioNum: decimal.NewFromInt(ratio), RatioDen: decimal.NewFromInt(1), OffsetDen: decimal.NewFromInt(1), IsBaseUnit: base, DimensionCode: "quantity"}
	}
	usd := domain.UnitFactors{RatioNum: decimal.NewFromInt(1), RatioDen: decimal.NewFromInt(1), OffsetDen: decimal.NewFromInt(1), IsBaseUnit: true, DimensionCode: "currency"}

	expectUnits := func(t *testing.T, factors map[string]domain.UnitFactors) domain.RepoFactory {
		ctrl := gomock.NewController(t)
		repos := factorymock.NewMockRepoFactory(ctrl)
		units := repositorymock.NewMockUnitConversionRepo(ctrl)
		repos.EXPECT().NewUnitConversionRepo().Return(units).Times(1)
		units.EXPECT().GetUnitFactors(gomock.Any(), "acct", gomock.Any()).Return(factors, nil).Times(1)
		return repos
	}

	t.Run("reads the units only for pairs that differ", func(t *testing.T) {
		repos := expectUnits(t, map[string]domain.UnitFactors{"unit_carton": count(24, false), "unit_pair": count(2, false), "unit_usd": usd})

		convs, apiErr := unitPairConversions(context.Background(), repos, "acct", []unitPair{carton, same})
		require.Nil(t, apiErr)
		require.Equal(t, "810", pricing.ExtendedPrice(decimal.NewFromInt(3), decimal.RequireFromString("22.5"), convs[carton]).String())
		require.Equal(t, pricing.Identity, convs[same])
	})

	t.Run("does not read units when every pair matches", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		convs, apiErr := unitPairConversions(context.Background(), factorymock.NewMockRepoFactory(ctrl), "acct", []unitPair{same})
		require.Nil(t, apiErr)
		require.Equal(t, pricing.Identity, convs[same])
	})

	t.Run("a unit it cannot find fails", func(t *testing.T) {
		repos := expectUnits(t, map[string]domain.UnitFactors{})
		_, apiErr := unitPairConversions(context.Background(), repos, "acct", []unitPair{carton})
		require.NotNil(t, apiErr)
	})

	t.Run("units the dashboard would price differently fail", func(t *testing.T) {
		mass := count(24, false)
		mass.DimensionCode = "mass"
		offset := count(2, false)
		offset.OffsetNum = decimal.NewFromInt(1)
		cents := usd
		cents.IsBaseUnit, cents.RatioDen = false, decimal.NewFromInt(100)
		for name, factors := range map[string]map[string]domain.UnitFactors{
			"different dimensions":   {"unit_carton": mass, "unit_pair": count(2, false), "unit_usd": usd},
			"an offset":              {"unit_carton": count(24, false), "unit_pair": offset, "unit_usd": usd},
			"a non-base currency":    {"unit_carton": count(24, false), "unit_pair": count(2, false), "unit_usd": cents},
			"a base unit not at one": {"unit_carton": count(24, true), "unit_pair": count(2, false), "unit_usd": usd},
		} {
			t.Run(name, func(t *testing.T) {
				repos := expectUnits(t, factors)
				_, apiErr := unitPairConversions(context.Background(), repos, "acct", []unitPair{carton})
				require.NotNil(t, apiErr)
			})
		}
	})
}
