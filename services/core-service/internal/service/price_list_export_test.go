package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/augno/api/shared/errors"
)

// The browser names a downloaded export from the last segment of its object key, so an export that is not a spreadsheet has to say so in the key. Getting this wrong ships a PDF named .xlsx, which refuses to open.
func TestExportObjectKey_ExtensionFollowsTheFormat(t *testing.T) {
	at := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		slug    string
		ext     string
		wantEnd string
	}{
		{"price list is a pdf", "price_list", "pdf", "price_list_export_08-14-2026.pdf"},
		{"spreadsheet exports are unchanged", "departments", "", "departments_export_08-14-2026.xlsx"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := exportObjectKey("ac_1", tc.slug, "job_1", at, tc.ext)
			if !strings.HasSuffix(key, tc.wantEnd) {
				t.Errorf("key = %q, want it to end in %q", key, tc.wantEnd)
			}
			if !strings.HasPrefix(key, "exports/ac_1/"+tc.slug+"/job_1/") {
				t.Errorf("key = %q, want it scoped to the account, resource and job", key)
			}
		})
	}
}

// priceListExportProduct is a product as the product export really hands it over: the product line is stitched on, but its unit group is not, so anything the document needs from the units has to be resolved by the price list itself.
func priceListExportProduct(id, sku, description, size string) *domain.ProductFull {
	lineID := "prln_1"
	return &domain.ProductFull{
		ID:            id,
		ProductLineID: &lineID,
		Item: &domain.Item{
			SKU:         sku,
			Description: &description,
			Attributes:  []*domain.ItemAttribute{{PropertyID: "prop_size", Value: size, Order: 1}},
		},
		ProductLine: &domain.ProductLineFull{ID: lineID, Name: "Couture", UnitGroupID: "ungr_1"},
	}
}

// A price list without a pack does not say what the customer is buying, and the units it is priced in are the same units the pack is expressed in — so losing the unit group empties the packing column and silently prices every line without its conversion. It went unnoticed once because the product export stitches the product line but not its units.
func TestBuildPriceListDocument_ResolvesTheUnitsTheProductExportLeavesOff(t *testing.T) {
	ctrl := gomock.NewController(t)
	products := []*domain.ProductFull{
		priceListExportProduct("prod_1", "820104", "Essence 15-20 mmHg Closed Toe Thigh", "A"),
		priceListExportProduct("prod_2", "820204", "Essence 15-20 mmHg Closed Toe Thigh", "B"),
	}

	customerRepo := repositorymock.NewMockCustomerRepo(ctrl)
	customerRepo.EXPECT().Get(gomock.Any(), "ac_1", "ac_2", gomock.Nil()).Return(&domain.Customer{Name: "Healthcare and Co"}, nil)

	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	accountRepo.EXPECT().GetByID(gomock.Any(), "ac_1").Return(&domain.Account{Name: "Seller Co"}, nil)

	productRepo := repositorymock.NewMockProductRepo(ctrl)
	productRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(products, nil)

	propertyRepo := repositorymock.NewMockPropertyRepo(ctrl)
	propertyRepo.EXPECT().GetByIDs(gomock.Any(), "ac_1", []string{"prop_size"}).Return([]*domain.Property{{ID: "prop_size", Name: "Size"}}, nil)

	// Both products share a line, so the group is worth resolving once however many products hang off it.
	productLineRepo := repositorymock.NewMockProductLineRepo(ctrl)
	productLineRepo.EXPECT().
		GetUnitGroup(gomock.Any(), "ac_1", "ungr_1", gomock.Any()).
		Return(&domain.ProductLineUnitGroup{
			ID:         "ungr_1",
			BaseUnitID: "unit_pair",
			BaseUnit:   &domain.LightUnit{ID: "unit_pair", Name: "Pair", Abbreviation: "pr", IsBaseUnit: true},
			AssociatedUnits: []*domain.UnitGroupUnit{
				{IsVisible: true, Unit: domain.LightUnit{ID: "unit_pair", Name: "Pair", IsBaseUnit: true, RatioNumerator: "1", RatioDenominator: "1"}},
				{IsVisible: true, Unit: domain.LightUnit{ID: "unit_carton", Name: "Carton", RatioNumerator: "10", RatioDenominator: "1"}},
			},
		}, nil).
		Times(1)

	// The bundle can only convert into a unit it was asked to load, which is the other half of what the missing group cost.
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
	repos.EXPECT().NewProductLineRepo().Return(productLineRepo).AnyTimes()
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
		t.Errorf("BaseUnitName = %q, want Pair — the cost column is headed with it", line.BaseUnitName)
	}
	if len(line.Sections) == 0 {
		t.Fatal("product line has no sections")
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
		t.Errorf("OrderedUnitIDs = %v, want the line's base unit", loaded.OrderedUnitIDs)
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
