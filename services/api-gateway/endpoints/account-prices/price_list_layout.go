package accountpriceep

import (
	"sort"
	"strings"
)

// priceListTier is one price column: the quantity at which the price was quoted.
type priceListTier struct {
	// Label shown in the column header, e.g. "1+" or "100+ pr".
	Label string
	// Quantity the price was quoted at.
	Quantity string
	// Unit the quantity is expressed in.
	UnitID           string
	UnitAbbreviation string
}

// priceListProduct is one SKU with everything the layout needs to place it.
type priceListProduct struct {
	ProductID   string
	SKU         string
	Description string
	// AttributeValues and AttributeOrders are keyed by property name. Orders carry the attribute's own sort order so sizes read S, M, L rather than L, M, S.
	AttributeValues map[string]string
	AttributeOrders map[string]int32
	// Prices aligned to the product line's tier list.
	Prices []string
	// Packing renders the sellable pack, e.g. "10 Pairs Per Carton".
	Packing string
}

// priceListRow is one table row: a SKU reduced to the section's column set.
type priceListRow struct {
	SKU         string
	Description string
	// Values aligned to the section's Columns.
	Values  []string
	Packing string
	// Prices aligned to the section's Tiers.
	Prices []string
}

// priceListSection is one table: the SKUs of a product line that share a price and a pack.
type priceListSection struct {
	// Heading names what every SKU in the section has in common, e.g. "Knee Length · Closed Toe".
	Heading string
	// Columns are the property names that vary inside the section, outermost first.
	Columns []string
	Tiers   []priceListTier
	Rows    []priceListRow
}

// priceListLine is one product line's worth of the document.
type priceListLine struct {
	ProductLineID   string
	ProductLineName string
	BaseUnitName    string
	Sections        []priceListSection
}

// buildPriceListSections groups one product line's SKUs into the document's tables.
//
// A section is a set of SKUs that share a price at every tier and share a pack — which is what a price list is really organized by, and why the price column ends up as a single merged cell. Within a section, properties whose value never changes describe the whole group and become the heading; the rest become columns.
func buildPriceListSections(products []priceListProduct, tiers []priceListTier) []priceListSection {
	if len(products) == 0 {
		return nil
	}

	// Partition on the price vector plus the pack, preserving first-seen order so the output is stable for a stable input.
	order := make([]string, 0)
	groups := make(map[string][]priceListProduct)
	for _, product := range products {
		key := strings.Join(product.Prices, "\x00") + "\x00|" + product.Packing
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], product)
	}

	sections := make([]priceListSection, 0, len(order))
	for _, key := range order {
		sections = append(sections, buildPriceListSection(groups[key], tiers))
	}

	// Cheapest first: a price list reads better ascending, and it makes the order independent of catalog insertion order.
	sort.SliceStable(sections, func(i, j int) bool {
		return sections[i].Heading < sections[j].Heading
	})
	return sections
}

// buildPriceListSection splits one price group's properties into heading and columns.
func buildPriceListSection(products []priceListProduct, tiers []priceListTier) priceListSection {
	propertyValues := make(map[string]map[string]struct{})
	for _, product := range products {
		for property, value := range product.AttributeValues {
			if propertyValues[property] == nil {
				propertyValues[property] = make(map[string]struct{})
			}
			propertyValues[property][value] = struct{}{}
		}
	}

	headingProperties := make([]string, 0)
	headingByProperty := make(map[string]string)
	columns := make([]string, 0)
	for property, values := range propertyValues {
		// A property only describes the whole group if every SKU carries it with the same value; one SKU missing it makes it a distinguishing column.
		if len(values) == 1 && countProductsWithProperty(products, property) == len(products) {
			headingProperties = append(headingProperties, property)
			for value := range values {
				headingByProperty[property] = value
			}
			continue
		}
		columns = append(columns, property)
	}
	// Ordered by property, not by value, so the heading of one section reads in the same order as the heading of the next.
	sort.Strings(headingProperties)
	headingParts := make([]string, 0, len(headingProperties))
	for _, property := range headingProperties {
		headingParts = append(headingParts, headingByProperty[property])
	}

	// Least-varying property outermost, so the widest merged spans sit on the left and each nested column subdivides the one before it.
	sort.SliceStable(columns, func(i, j int) bool {
		li, lj := len(propertyValues[columns[i]]), len(propertyValues[columns[j]])
		if li != lj {
			return li < lj
		}
		return columns[i] < columns[j]
	})

	sorted := make([]priceListProduct, len(products))
	copy(sorted, products)
	sort.SliceStable(sorted, func(i, j int) bool {
		for _, column := range columns {
			oi, oj := sorted[i].AttributeOrders[column], sorted[j].AttributeOrders[column]
			if oi != oj {
				return oi < oj
			}
			vi, vj := sorted[i].AttributeValues[column], sorted[j].AttributeValues[column]
			if vi != vj {
				return vi < vj
			}
		}
		return sorted[i].SKU < sorted[j].SKU
	})

	rows := make([]priceListRow, 0, len(sorted))
	for _, product := range sorted {
		values := make([]string, len(columns))
		for i, column := range columns {
			values[i] = product.AttributeValues[column]
		}
		rows = append(rows, priceListRow{
			SKU:         product.SKU,
			Description: product.Description,
			Values:      values,
			Packing:     product.Packing,
			Prices:      product.Prices,
		})
	}

	section := priceListSection{
		Heading: strings.Join(headingParts, " · "),
		Columns: columns,
		Tiers:   tiers,
		Rows:    rows,
	}
	return dropFlatTiers(section)
}

// countProductsWithProperty counts the SKUs carrying a value for the property.
func countProductsWithProperty(products []priceListProduct, property string) int {
	n := 0
	for _, product := range products {
		if _, ok := product.AttributeValues[property]; ok {
			n++
		}
	}
	return n
}

// dropFlatTiers removes price columns that never differ from the column before them, so a section with no volume break prints one price column instead of five identical ones.
func dropFlatTiers(section priceListSection) priceListSection {
	if len(section.Tiers) <= 1 || len(section.Rows) == 0 {
		return section
	}

	keep := []int{0}
	for i := 1; i < len(section.Tiers); i++ {
		previous := keep[len(keep)-1]
		for _, row := range section.Rows {
			if i < len(row.Prices) && previous < len(row.Prices) && row.Prices[i] != row.Prices[previous] {
				keep = append(keep, i)
				break
			}
		}
	}
	if len(keep) == len(section.Tiers) {
		return section
	}

	tiers := make([]priceListTier, 0, len(keep))
	for _, i := range keep {
		tiers = append(tiers, section.Tiers[i])
	}
	rows := make([]priceListRow, len(section.Rows))
	for r, row := range section.Rows {
		prices := make([]string, 0, len(keep))
		for _, i := range keep {
			if i < len(row.Prices) {
				prices = append(prices, row.Prices[i])
			}
		}
		row.Prices = prices
		rows[r] = row
	}

	section.Tiers = tiers
	section.Rows = rows
	return section
}

// mergeSpans returns, for each row, how many rows the cell at that position spans: n for the first row of a run of equal values, 0 for the rows it swallows.
func mergeSpans(values []string) []int {
	spans := make([]int, len(values))
	i := 0
	for i < len(values) {
		j := i + 1
		for j < len(values) && values[j] == values[i] {
			j++
		}
		spans[i] = j - i
		i = j
	}
	return spans
}

// mergeSpansNested computes the vertical merge spans of every attribute column at once. A run may only swallow rows that also merge in each column to its left — without that, an inner "Regular" run would visually span the boundary between an outer Black block and the Khaki block below it.
//
// rows is indexed [row][column]; the result is indexed the same way.
func mergeSpansNested(rows [][]string, columns int) [][]int {
	spans := make([][]int, len(rows))
	for r := range spans {
		spans[r] = make([]int, columns)
	}
	if len(rows) == 0 || columns == 0 {
		return spans
	}

	// breaks[r] reports whether row r starts a new run in some column to the left of the one being measured. Seeded true at row 0 so every column opens a run there.
	breaks := make([]bool, len(rows))
	breaks[0] = true

	for c := 0; c < columns; c++ {
		start := 0
		for r := 1; r <= len(rows); r++ {
			atEnd := r == len(rows)
			if !atEnd && !breaks[r] && rows[r][c] == rows[start][c] {
				continue
			}
			spans[start][c] = r - start
			if !atEnd {
				start = r
			}
		}
		// Rows that opened a run in this column are boundaries for the columns to its right.
		for r := range rows {
			if spans[r][c] > 0 {
				breaks[r] = true
			}
		}
	}
	return spans
}
