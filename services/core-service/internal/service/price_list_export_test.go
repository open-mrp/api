package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// The browser names a downloaded export from the last segment of its object key, so an export that is not a spreadsheet has to say so in the key. Getting this wrong ships a PDF named .xlsx, which refuses to open.
func TestExportObjectKey_ExtensionFollowsTheFormat(t *testing.T) {
	at := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		slug     string
		ext      string
		fileName string
		wantEnd  string
	}{
		{"price list is a pdf", "price_list", "pdf", "", "price_list_export_08-14-2026.pdf"},
		{"spreadsheet exports are unchanged", "departments", "", "", "departments_export_08-14-2026.xlsx"},
		{"an export named for its filters keeps that name", "inventory_change_logs", "", "inventory-change-logs-all-all.xlsx", "/inventory-change-logs-all-all.xlsx"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := exportObjectKey("ac_1", "job_1", at, exportJobPayload{Slug: tc.slug, Ext: tc.ext, Name: tc.fileName})
			if !strings.HasSuffix(key, tc.wantEnd) {
				t.Errorf("key = %q, want it to end in %q", key, tc.wantEnd)
			}
			if !strings.HasPrefix(key, "exports/ac_1/"+tc.slug+"/job_1/") {
				t.Errorf("key = %q, want it scoped to the account, resource and job", key)
			}
		})
	}
}

// priceListExportProduct is a product as the product export hands it over: its item and product line, but not its
// units, which the price list reads from the item's own stocking and order units.
func priceListExportProduct(id, sku, description, size string) *domain.ProductFull {
	lineID := "prln_1"
	return &domain.ProductFull{
		ID:            id,
		ItemID:        "it_" + id,
		ProductLineID: &lineID,
		Item: &domain.Item{
			ID:          "it_" + id,
			SKU:         sku,
			Description: &description,
			Attributes:  []*domain.ItemAttribute{{PropertyID: "prop_size", Value: size, Order: 1}},
		},
		ProductLine: &domain.ProductLineFull{ID: lineID, Name: "Classic"},
	}
}

var (
	plUnitEach   = &domain.Unit{ID: "unit_each", Name: "Each", IsBaseUnit: true, RatioNumerator: "1", RatioDenominator: "1"}
	plUnitPair   = &domain.Unit{ID: "unit_pair", Name: "Pair", Abbreviation: "pr", RatioNumerator: "2", RatioDenominator: "1"}
	plUnitCarton = &domain.Unit{ID: "unit_carton", Name: "Carton", RatioNumerator: "20", RatioDenominator: "1"}
	plUnitDozen  = &domain.Unit{ID: "unit_dozen", Name: "Dozen", RatioNumerator: "12", RatioDenominator: "1"}
)

// Each product is quoted per its item's default order unit, and its packing names the largest unit customers can order
// it in, both read from the item's own units.
func TestBuildPriceListDocument_QuotesEachItemInItsOwnUnits(t *testing.T) {
	ctrl := gomock.NewController(t)
	products := []*domain.ProductFull{
		priceListExportProduct("prod_1", "SK-1001", "Classic Crew Sock", "A"),
		priceListExportProduct("prod_2", "SK-1002", "Classic Crew Sock", "B"),
	}

	customerRepo := repositorymock.NewMockCustomerRepo(ctrl)
	customerRepo.EXPECT().Get(gomock.Any(), "ac_1", "ac_2", gomock.Nil()).Return(&domain.Customer{Name: "Healthcare and Co"}, nil)

	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	accountRepo.EXPECT().GetByID(gomock.Any(), "ac_1").Return(&domain.Account{Name: "Seller Co"}, nil)

	productRepo := repositorymock.NewMockProductRepo(ctrl)
	productRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(products, nil)

	propertyRepo := repositorymock.NewMockPropertyRepo(ctrl)
	propertyRepo.EXPECT().GetByIDs(gomock.Any(), "ac_1", []string{"prop_size"}).Return([]*domain.Property{{ID: "prop_size", Name: "Size"}}, nil)

	units := &domain.ItemUnits{StockingUnitID: "unit_each", DefaultOrderUnitID: "unit_pair", OrderUnits: []domain.ItemOrderUnit{
		{UnitID: "unit_pair", PortalVisibility: constants.CustomerPortalVisibilityVisible},
		{UnitID: "unit_carton", PortalVisibility: constants.CustomerPortalVisibilityVisible},
	}}
	itemUnitsRepo := repositorymock.NewMockItemUnitsRepo(ctrl)
	itemUnitsRepo.EXPECT().Get(gomock.Any(), []string{"it_prod_1", "it_prod_2"}).
		Return(map[string]*domain.ItemUnits{"it_prod_1": units, "it_prod_2": units}, nil)

	unitRepo := repositorymock.NewMockUnitRepo(ctrl)
	unitRepo.EXPECT().GetByIDs(gomock.Any(), "ac_1", gomock.Any()).Return([]*domain.Unit{plUnitEach, plUnitPair, plUnitCarton}, nil)

	var loaded domain.LoadPricingBundleParams
	pricingRepo := repositorymock.NewMockPricingRepo(ctrl)
	pricingRepo.EXPECT().
		LoadPricingBundle(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.LoadPricingBundleParams) (*domain.PricingBundle, *apierror.APIError) {
			loaded = params
			return &domain.PricingBundle{}, nil
		})

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewCustomerRepo().Return(customerRepo).AnyTimes()
	repos.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
	repos.EXPECT().NewProductRepo().Return(productRepo).AnyTimes()
	repos.EXPECT().NewPropertyRepo().Return(propertyRepo).AnyTimes()
	repos.EXPECT().NewItemUnitsRepo().Return(itemUnitsRepo).AnyTimes()
	repos.EXPECT().NewUnitRepo().Return(unitRepo).AnyTimes()
	repos.EXPECT().NewPricingRepo().Return(pricingRepo).AnyTimes()

	svc := &accountPriceSvcImpl{repos: repos}
	doc, rows, apiErr := svc.buildPriceListDocument(context.Background(), "ac_1", priceListExportFilters{CustomerAccountID: "ac_2"})
	if apiErr != nil {
		t.Fatalf("buildPriceListDocument: %v", apiErr)
	}
	if rows != 2 {
		t.Errorf("rows = %d, want 2", rows)
	}
	if len(doc.Lines) != 1 {
		t.Fatalf("document has %d product lines, want 1", len(doc.Lines))
	}

	line := doc.Lines[0]
	if line.BaseUnitName != "Pair" {
		t.Errorf("BaseUnitName = %q, want Pair, the items' default order unit", line.BaseUnitName)
	}
	for _, section := range line.Sections {
		for _, row := range section.Rows {
			if row.Packing != "10 Pairs Per Carton" {
				t.Errorf("%s packing = %q, want 10 Pairs Per Carton", row.SKU, row.Packing)
			}
		}
		for _, tier := range section.Tiers {
			if header := plCostHeader(line, section, tier); header != "Cost Per Pair" {
				t.Errorf("cost header = %q, want Cost Per Pair", header)
			}
		}
	}
	if len(loaded.OrderedUnitIDs) != 1 || loaded.OrderedUnitIDs[0] != "unit_pair" {
		t.Errorf("OrderedUnitIDs = %v, want the default order unit", loaded.OrderedUnitIDs)
	}
}

// The price list registers itself as a PDF; an empty Ext here would silently produce a spreadsheet name.
func TestPriceListExportSpec_DeclaresPDF(t *testing.T) {
	spec := (&accountPriceSvcImpl{}).priceListExportSpec()

	if spec.Ext != "pdf" {
		t.Errorf("Ext = %q, want pdf", spec.Ext)
	}
	if spec.Slug != "price_list" {
		t.Errorf("Slug = %q, want price_list", spec.Slug)
	}
}

func priceListTestCatalog(units *domain.ItemUnits, all ...*domain.Unit) *priceListCatalog {
	c := &priceListCatalog{itemUnits: map[string]*domain.ItemUnits{"it_1": units}, units: map[string]*domain.LightUnit{}}
	for _, u := range all {
		c.units[u.ID] = &domain.LightUnit{ID: u.ID, Name: u.Name, RatioNumerator: u.RatioNumerator, RatioDenominator: u.RatioDenominator}
	}
	return c
}

// Unit ratios are stored against the base of the dimension (Each), so a pack is measured against the unit the item is
// quoted per, not against Each; an order unit that holds its own quantity of stocking units uses that quantity.
func TestPriceListPacking_MeasuresThePackAgainstTheQuoteUnit(t *testing.T) {
	product := &domain.ProductFull{ItemID: "it_1"}
	visible := constants.CustomerPortalVisibilityVisible
	cases := []struct {
		name    string
		catalog *priceListCatalog
		want    string
	}{
		{
			name: "a pack above the quote unit counts quote units",
			catalog: priceListTestCatalog(&domain.ItemUnits{StockingUnitID: "unit_each", DefaultOrderUnitID: "unit_pair", OrderUnits: []domain.ItemOrderUnit{
				{UnitID: "unit_pair", PortalVisibility: visible}, {UnitID: "unit_dozen", PortalVisibility: visible},
			}}, plUnitEach, plUnitPair, plUnitDozen),
			want: "6 Pairs Per Dozen",
		},
		{
			name: "an item's own pack quantity overrides the unit's ratio",
			catalog: priceListTestCatalog(&domain.ItemUnits{StockingUnitID: "unit_each", DefaultOrderUnitID: "unit_pair", OrderUnits: []domain.ItemOrderUnit{
				{UnitID: "unit_pair", PortalVisibility: visible}, {UnitID: "unit_carton", PortalVisibility: visible, StockingQuantity: new("24")},
			}}, plUnitEach, plUnitPair, plUnitCarton),
			want: "12 Pairs Per Carton",
		},
		{
			name: "a hidden pack is not offered",
			catalog: priceListTestCatalog(&domain.ItemUnits{StockingUnitID: "unit_each", DefaultOrderUnitID: "unit_pair", OrderUnits: []domain.ItemOrderUnit{
				{UnitID: "unit_pair", PortalVisibility: visible}, {UnitID: "unit_dozen", PortalVisibility: constants.CustomerPortalVisibilityHidden},
			}}, plUnitEach, plUnitPair, plUnitDozen),
			want: "Pair",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.catalog.packing(product); got != tc.want {
				t.Errorf("packing = %q, want %q", got, tc.want)
			}
		})
	}
}
