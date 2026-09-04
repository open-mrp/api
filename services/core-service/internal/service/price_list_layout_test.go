package service

import (
	"reflect"
	"testing"
)

func product(sku, packing string, prices []string, attrs map[string]string, orders map[string]int32) priceListProduct {
	return priceListProduct{
		ProductID:       sku,
		SKU:             sku,
		Description:     sku + " description",
		AttributeValues: attrs,
		AttributeOrders: orders,
		Prices:          prices,
		Packing:         packing,
	}
}

var oneTier = []priceListTier{{Label: "1+", Quantity: "1"}}

// A property every SKU shares describes the section; the rest become columns.
func TestBuildPriceListSections_ConstantPropertyBecomesHeading(t *testing.T) {
	products := []priceListProduct{
		product("510", "10 Pairs Per Carton", []string{"4.10"},
			map[string]string{"Length": "Knee", "Size": "Small"}, map[string]int32{"Size": 1}),
		product("511", "10 Pairs Per Carton", []string{"4.10"},
			map[string]string{"Length": "Knee", "Size": "Medium"}, map[string]int32{"Size": 2}),
	}

	sections := buildPriceListSections(products, oneTier)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0].Heading != "Knee" {
		t.Errorf("heading = %q, want %q", sections[0].Heading, "Knee")
	}
	if !reflect.DeepEqual(sections[0].Columns, []string{"Size"}) {
		t.Errorf("columns = %v, want [Size]", sections[0].Columns)
	}
}

// Differing prices split the catalog into separate tables, which is what makes the price column a single merged cell per section.
func TestBuildPriceListSections_SplitsOnPrice(t *testing.T) {
	products := []priceListProduct{
		product("510", "10 Pairs Per Carton", []string{"4.10"},
			map[string]string{"Length": "Knee", "Size": "Small"}, map[string]int32{"Size": 1}),
		product("511", "10 Pairs Per Carton", []string{"4.10"},
			map[string]string{"Length": "Knee", "Size": "Medium"}, map[string]int32{"Size": 2}),
		product("610", "10 Pairs Per Carton", []string{"5.95"},
			map[string]string{"Length": "Thigh", "Size": "Small"}, map[string]int32{"Size": 1}),
		product("611", "10 Pairs Per Carton", []string{"5.95"},
			map[string]string{"Length": "Thigh", "Size": "Medium"}, map[string]int32{"Size": 2}),
	}

	sections := buildPriceListSections(products, oneTier)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Heading != "Knee" || sections[1].Heading != "Thigh" {
		t.Errorf("headings = %q, %q", sections[0].Heading, sections[1].Heading)
	}
	for _, section := range sections {
		if !reflect.DeepEqual(section.Columns, []string{"Size"}) {
			t.Errorf("columns = %v, want [Size]", section.Columns)
		}
		if len(section.Rows) != 2 {
			t.Errorf("rows = %d, want 2", len(section.Rows))
		}
	}
}

// Every property being constant is legitimate: a one-SKU section carries all of its detail in the heading and needs no attribute columns at all.
func TestBuildPriceListSection_SingleProductHasNoColumns(t *testing.T) {
	products := []priceListProduct{
		product("510", "pack", []string{"4.10"},
			map[string]string{"Length": "Knee", "Size": "Small"}, nil),
	}

	section := buildPriceListSection(products, oneTier)
	if len(section.Columns) != 0 {
		t.Errorf("columns = %v, want none", section.Columns)
	}
	if section.Heading != "Knee · Small" {
		t.Errorf("heading = %q, want %q", section.Heading, "Knee · Small")
	}
}

// Rows order by the attribute's own sort order, so sizes read S, M, L.
func TestBuildPriceListSection_OrdersRowsByAttributeSortOrder(t *testing.T) {
	products := []priceListProduct{
		product("c", "pack", []string{"1.00"}, map[string]string{"Size": "Large"}, map[string]int32{"Size": 3}),
		product("a", "pack", []string{"1.00"}, map[string]string{"Size": "Small"}, map[string]int32{"Size": 1}),
		product("b", "pack", []string{"1.00"}, map[string]string{"Size": "Medium"}, map[string]int32{"Size": 2}),
	}

	section := buildPriceListSection(products, oneTier)
	got := []string{section.Rows[0].Values[0], section.Rows[1].Values[0], section.Rows[2].Values[0]}
	want := []string{"Small", "Medium", "Large"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("row order = %v, want %v", got, want)
	}
}

// The least-varying property sits leftmost so the widest merged spans nest outermost.
func TestBuildPriceListSection_OrdersColumnsByCardinality(t *testing.T) {
	products := []priceListProduct{
		product("1", "pack", []string{"18.00"}, map[string]string{"Color": "Black", "Size": "A", "Toe": "Closed"}, nil),
		product("2", "pack", []string{"18.00"}, map[string]string{"Color": "Black", "Size": "B", "Toe": "Closed"}, nil),
		product("3", "pack", []string{"18.00"}, map[string]string{"Color": "Khaki", "Size": "A", "Toe": "Open"}, nil),
		product("4", "pack", []string{"18.00"}, map[string]string{"Color": "Khaki", "Size": "C", "Toe": "Open"}, nil),
	}

	section := buildPriceListSection(products, oneTier)
	// Color and Toe have two values each, Size has three, so the two-value columns come first and the alphabetical tie-break puts Color ahead of Toe.
	want := []string{"Color", "Toe", "Size"}
	if !reflect.DeepEqual(section.Columns, want) {
		t.Errorf("columns = %v, want %v", section.Columns, want)
	}
}

// A tier column that never changes the price is dropped, so a line with no volume break prints one price column rather than several identical ones.
func TestDropFlatTiers(t *testing.T) {
	tiers := []priceListTier{{Label: "1+"}, {Label: "100+"}, {Label: "500+"}}
	products := []priceListProduct{
		product("a", "pack", []string{"4.10", "4.10", "3.90"}, map[string]string{"Size": "S"}, nil),
		product("b", "pack", []string{"4.10", "4.10", "3.90"}, map[string]string{"Size": "M"}, nil),
	}

	section := buildPriceListSection(products, tiers)
	if len(section.Tiers) != 2 {
		t.Fatalf("expected 2 tiers after dropping the flat one, got %d", len(section.Tiers))
	}
	if section.Tiers[0].Label != "1+" || section.Tiers[1].Label != "500+" {
		t.Errorf("kept tiers = %q, %q", section.Tiers[0].Label, section.Tiers[1].Label)
	}
	if !reflect.DeepEqual(section.Rows[0].Prices, []string{"4.10", "3.90"}) {
		t.Errorf("prices = %v, want [4.10 3.90]", section.Rows[0].Prices)
	}
}

func TestMergeSpans(t *testing.T) {
	got := mergeSpans([]string{"a", "a", "b", "c", "c", "c"})
	want := []int{2, 0, 1, 3, 0, 0}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("spans = %v, want %v", got, want)
	}
}

// An inner column must not merge across a boundary opened by an outer column.
func TestMergeSpansNested_InnerRunStopsAtOuterBoundary(t *testing.T) {
	rows := [][]string{
		{"Black", "Regular"},
		{"Black", "Regular"},
		{"Khaki", "Regular"},
		{"Khaki", "Regular"},
	}

	spans := mergeSpansNested(rows, 2)

	if spans[0][0] != 2 || spans[2][0] != 2 {
		t.Errorf("outer spans = %d, %d; want 2, 2", spans[0][0], spans[2][0])
	}
	// Without the nesting rule this would be a single span of 4 crossing the color change.
	if spans[0][1] != 2 || spans[2][1] != 2 {
		t.Errorf("inner spans = %d, %d; want 2, 2", spans[0][1], spans[2][1])
	}
	if spans[1][1] != 0 || spans[3][1] != 0 {
		t.Errorf("swallowed rows = %d, %d; want 0, 0", spans[1][1], spans[3][1])
	}
}
