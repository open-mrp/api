package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/safeconv"
)

// maxPriceListTierColumns bounds how many volume breaks become price columns. Past this the table stops being readable, and the ones dropped are disclosed on the title page rather than silently omitted.
const maxPriceListTierColumns = 4

// priceListExportFilters is what the accept records on the job. The render re-reads it rather than the identity it no longer has.
type priceListExportFilters struct {
	CustomerAccountID string `json:"customer_account_id"`
}

// priceListExportSpec wires the price list into the export engine. It carries no columns because the document is a PDF rather than a sheet — only the engine's accept path (permission, slug, format) is shared.
func (s *accountPriceSvcImpl) priceListExportSpec() exportSpec[*domain.ProductFull, priceListExportFilters] {
	return exportSpec[*domain.ProductFull, priceListExportFilters]{
		PermissionDomain: types.PermissionDomainDiscounts,
		Name:             "Price List",
		Slug:             "price_list",
		ResourceType:     constants.ObjectTypeProduct,
		Ext:              "pdf",
	}
}

// ExportPriceList accepts a price-list export: it records the customer on a job and returns it to poll. Nothing is rendered here — pricing a whole catalog is far too slow to hold a request open for.
func (s *accountPriceSvcImpl) ExportPriceList(ctx context.Context, params domain.ExportPriceListParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.priceListExportSpec(), priceListExportFilters{
		CustomerAccountID: params.CustomerAccountID,
	})
}

// BuildExportPriceList renders the PDF an accepted export recorded.
func (s *accountPriceSvcImpl) BuildExportPriceList(ctx context.Context, accountID string, raw json.RawMessage) (*domain.Export, *apierror.APIError) {
	var filters priceListExportFilters
	if err := json.Unmarshal(raw, &filters); err != nil {
		return nil, apierror.NewInternalError(err, "Job items are not a price list export payload.")
	}
	if filters.CustomerAccountID == "" {
		return nil, apierror.NewInvariantViolationError("A price list export is missing its customer.")
	}

	doc, rows, apiErr := s.buildPriceListDocument(ctx, accountID, filters)
	if apiErr != nil {
		return nil, apiErr
	}

	body, err := buildPriceListPDF(doc)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to render the price list.")
	}
	return &domain.Export{ContentType: "application/pdf", Body: body, RowCount: rows}, nil
}

// buildPriceListDocument gathers everything the document states, and how many products it covers.
//
// Every price here comes from the same engine that prices a sales order, so the document cannot drift from what the customer is actually charged. The pricing bundle is loaded once for the whole catalog and every product priced against it in memory — the alternative, one priced batch per request, is what made this too slow to run synchronously.
func (s *accountPriceSvcImpl) buildPriceListDocument(ctx context.Context, accountID string, filters priceListExportFilters) (priceListDocument, int32, *apierror.APIError) {
	doc := priceListDocument{DateLong: time.Now().UTC().Format("January 2, 2006")}

	customer, apiErr := s.repos.NewCustomerRepo().Get(ctx, accountID, filters.CustomerAccountID, nil)
	if apiErr != nil {
		return doc, 0, apiErr
	}
	doc.CustomerName = customer.Name
	if customer.DefaultPaymentTermName != nil {
		doc.PaymentTerm = *customer.DefaultPaymentTermName
	}
	if customer.DefaultShippingTermName != nil {
		doc.ShippingTerm = *customer.DefaultShippingTermName
	}

	// Letterhead is best-effort: a price list without a logo is still a price list.
	if account, _ := s.repos.NewAccountRepo().GetByID(ctx, accountID); account != nil {
		doc.MerchantName = account.Name
		if account.Branding != nil && account.Branding.LogoURL != nil {
			doc.LogoImageType, doc.LogoImage = fetchAckLogoImage(ctx, *account.Branding.LogoURL)
		}
	}

	// The customer filter applies the same three product-line visibility pathways order entry uses, so the document can never quote something they cannot order.
	products, apiErr := s.repos.NewProductRepo().Export(ctx, domain.ExportProductsParams{
		AccountID:   accountID,
		CustomerIDs: []string{filters.CustomerAccountID},
	})
	if apiErr != nil {
		return doc, 0, apiErr
	}
	products = priceListSellableProducts(products)
	if len(products) == 0 {
		doc.Notes = []string{"No products are currently available to this customer."}
		return doc, 0, nil
	}
	if apiErr := s.hydratePriceListUnitGroups(ctx, accountID, products); apiErr != nil {
		return doc, 0, apiErr
	}

	propertyNames, apiErr := s.priceListPropertyNames(ctx, accountID, products)
	if apiErr != nil {
		return doc, 0, apiErr
	}

	bundle, apiErr := s.repos.NewPricingRepo().LoadPricingBundle(ctx, domain.LoadPricingBundleParams{
		OwnerAccountID: accountID,
		BuyerAccountID: filters.CustomerAccountID,
		ProductIDs:     priceListProductIDs(products),
		OrderedUnitIDs: priceListBaseUnitIDs(products),
	})
	if apiErr != nil {
		return doc, 0, apiErr
	}

	tiers, dropped := priceListTiersFromBundle(bundle, products)
	if dropped > 0 {
		doc.Notes = append(doc.Notes, fmt.Sprintf("%d additional volume breaks are not shown.", dropped))
	}

	doc.Lines = assemblePriceListLines(products, priceListPrices(bundle, products, tiers), propertyNames, tiers)
	return doc, safeconv.IntToInt32(len(products)), nil
}

// priceListSellableProducts drops products with no product line: they can never match an account price or a line-scoped discount, and have no unit group to price against.
func priceListSellableProducts(products []*domain.ProductFull) []*domain.ProductFull {
	out := make([]*domain.ProductFull, 0, len(products))
	for _, product := range products {
		if product.ProductLine != nil && product.Item != nil {
			out = append(out, product)
		}
	}
	return out
}

// hydratePriceListUnitGroups attaches each product line's unit group, resolved once per group rather than once per line.
//
// The product export stitches the line but not its units, and without them the document has no pack to quote, no base unit to price against and no name for the cost column.
func (s *accountPriceSvcImpl) hydratePriceListUnitGroups(ctx context.Context, accountID string, products []*domain.ProductFull) *apierror.APIError {
	repo := s.repos.NewProductLineRepo()
	groups := make(map[string]*domain.ProductLineUnitGroup)
	for _, product := range products {
		unitGroupID := product.ProductLine.UnitGroupID
		if unitGroupID == "" {
			continue
		}
		if _, loaded := groups[unitGroupID]; !loaded {
			group, apiErr := repo.GetUnitGroup(ctx, accountID, unitGroupID, []string{"unit_group.base_unit", "unit_group.associated_units"})
			if apiErr != nil {
				return apiErr
			}
			groups[unitGroupID] = group
		}
		product.ProductLine.UnitGroup = groups[unitGroupID]
	}
	return nil
}

func priceListProductIDs(products []*domain.ProductFull) []string {
	ids := make([]string, 0, len(products))
	for _, product := range products {
		ids = append(ids, product.ID)
	}
	return ids
}

// priceListBaseUnitIDs is every per-unit basis the catalog is priced on, which is what the bundle needs to convert into.
func priceListBaseUnitIDs(products []*domain.ProductFull) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, product := range products {
		id := priceListBaseUnitID(product)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func priceListBaseUnitID(product *domain.ProductFull) string {
	if product.ProductLine == nil || product.ProductLine.UnitGroup == nil {
		return ""
	}
	return product.ProductLine.UnitGroup.BaseUnitID
}

// priceListPropertyNames resolves the property behind each attribute, since the table needs column headings and the product rows carry only ids.
func (s *accountPriceSvcImpl) priceListPropertyNames(ctx context.Context, accountID string, products []*domain.ProductFull) (map[string]string, *apierror.APIError) {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, product := range products {
		for _, attribute := range product.Item.Attributes {
			if _, ok := seen[attribute.PropertyID]; ok || attribute.PropertyID == "" {
				continue
			}
			seen[attribute.PropertyID] = struct{}{}
			ids = append(ids, attribute.PropertyID)
		}
	}
	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	properties, apiErr := s.repos.NewPropertyRepo().GetByIDs(ctx, accountID, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	names := make(map[string]string, len(properties))
	for _, property := range properties {
		names[property.ID] = property.Name
	}
	return names, nil
}

// priceListUnit reads a unit off the catalog, since the pricing bundle carries only conversion factors and not the names a column has to be headed with.
func priceListUnit(products []*domain.ProductFull, unitID string) *domain.LightUnit {
	if unitID == "" {
		return nil
	}
	for _, product := range products {
		group := product.ProductLine.UnitGroup
		if group == nil {
			continue
		}
		if group.BaseUnit != nil && group.BaseUnit.ID == unitID {
			return group.BaseUnit
		}
		for _, associated := range group.AssociatedUnits {
			if associated.Unit.ID == unitID {
				return &associated.Unit
			}
		}
	}
	return nil
}

// priceListTiersFromBundle builds the candidate price columns: quantity 1, then each distinct volume-discount threshold the customer can reach. The bundle already filtered the discounts to this buyer, so no scoping is re-derived here.
func priceListTiersFromBundle(bundle *domain.PricingBundle, products []*domain.ProductFull) ([]priceListTier, int) {
	baseUnitID := priceListDominantBaseUnit(products)
	tiers := []priceListTier{priceListTierFor("1", baseUnitID, priceListUnit(products, baseUnitID))}

	type candidate struct {
		quantity decimal.Decimal
		unitID   string
		unit     *domain.LightUnit
	}
	seen := make(map[string]struct{})
	candidates := make([]candidate, 0)
	for _, discount := range bundle.VolumeDiscounts {
		if len(discount.AcceptableUnitIDs) == 0 {
			continue
		}
		unitID := discount.AcceptableUnitIDs[0]
		unit := priceListUnit(products, unitID)
		for _, tier := range discount.Tiers {
			threshold, err := decimal.NewFromString(tier.Threshold)
			if err != nil || threshold.LessThanOrEqual(decimal.NewFromInt(1)) {
				continue
			}
			key := threshold.String() + "\x00" + unitID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, candidate{quantity: threshold, unitID: unitID, unit: unit})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].quantity.LessThan(candidates[j].quantity) })

	dropped := 0
	if len(candidates) > maxPriceListTierColumns-1 {
		dropped = len(candidates) - (maxPriceListTierColumns - 1)
		candidates = candidates[:maxPriceListTierColumns-1]
	}
	for _, c := range candidates {
		tiers = append(tiers, priceListTierFor(c.quantity.String(), c.unitID, c.unit))
	}
	return tiers, dropped
}

// priceListDominantBaseUnit picks the per-unit basis most of the catalog is priced on, used for the quantity-1 column.
func priceListDominantBaseUnit(products []*domain.ProductFull) string {
	counts := make(map[string]int)
	for _, product := range products {
		if id := priceListBaseUnitID(product); id != "" {
			counts[id]++
		}
	}
	bestID, bestCount := "", 0
	for id, count := range counts {
		if count > bestCount || (count == bestCount && id < bestID) {
			bestID, bestCount = id, count
		}
	}
	return bestID
}

// priceListTierFor labels one price column. The quantity is what the price was quoted at; the unit is what the price under it is per, which is what the column has to say so a number without a basis is never printed.
func priceListTierFor(quantity, unitID string, unit *domain.LightUnit) priceListTier {
	tier := priceListTier{Label: quantity + "+", Quantity: quantity, UnitID: unitID}
	if unit != nil {
		tier.UnitName = unit.Name
		tier.UnitAbbreviation = unit.Abbreviation
		tier.Label = quantity + "+ " + unit.Abbreviation
	}
	return tier
}

// priceListPrices prices every product at every tier against a bundle that was loaded once.
//
// Each product is priced as a line on its own rather than as part of one big order: a price list quotes what this SKU costs at this quantity, whereas the volume-discount stage sums quantities across every line sharing a discount and would otherwise let unrelated products inflate each other's break.
func priceListPrices(bundle *domain.PricingBundle, products []*domain.ProductFull, tiers []priceListTier) map[string][]string {
	priced := make(map[string][]string, len(products))
	for _, product := range products {
		row := make([]string, len(tiers))
		for t, tier := range tiers {
			unitID := tier.UnitID
			if unitID == "" {
				unitID = priceListBaseUnitID(product)
			}
			line := domain.SalesOrderPriceLineInput{
				ProductID:      product.ID,
				QuantityValue:  tier.Quantity,
				QuantityUnitID: unitID,
			}
			price := computeUnitPrice(bundle, line, []domain.SalesOrderPriceLineInput{line})
			row[t] = formatPriceListMoney(price.Value)
		}
		priced[product.ID] = row
	}
	return priced
}

func formatPriceListMoney(value string) string {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return ""
	}
	return "$" + amount.StringFixed(2)
}

// assemblePriceListLines turns the priced catalog into one document section per product line.
func assemblePriceListLines(
	products []*domain.ProductFull,
	priced map[string][]string,
	propertyNames map[string]string,
	tiers []priceListTier,
) []priceListLine {
	type bucket struct {
		name         string
		baseUnitName string
		packing      string
		products     []priceListProduct
	}
	buckets := make(map[string]*bucket)
	order := make([]string, 0)

	for _, product := range products {
		line := product.ProductLine
		b, ok := buckets[line.ID]
		if !ok {
			b = &bucket{name: line.Name, packing: priceListPacking(line.UnitGroup)}
			if line.UnitGroup != nil && line.UnitGroup.BaseUnit != nil {
				b.baseUnitName = line.UnitGroup.BaseUnit.Name
			}
			buckets[line.ID] = b
			order = append(order, line.ID)
		}

		values := make(map[string]string)
		orders := make(map[string]int32)
		for _, attribute := range product.Item.Attributes {
			name := propertyNames[attribute.PropertyID]
			if name == "" {
				continue
			}
			values[name] = attribute.Value
			orders[name] = attribute.Order
		}

		description := ""
		if product.Item.Description != nil {
			description = *product.Item.Description
		}

		b.products = append(b.products, priceListProduct{
			ProductID:       product.ID,
			SKU:             product.Item.SKU,
			Description:     description,
			AttributeValues: values,
			AttributeOrders: orders,
			Prices:          priced[product.ID],
			Packing:         b.packing,
		})
	}

	sort.SliceStable(order, func(i, j int) bool { return buckets[order[i]].name < buckets[order[j]].name })

	lines := make([]priceListLine, 0, len(order))
	for _, id := range order {
		b := buckets[id]
		lines = append(lines, priceListLine{
			ProductLineID:   id,
			ProductLineName: b.name,
			BaseUnitName:    b.baseUnitName,
			Sections:        buildPriceListSections(b.products, tiers),
		})
	}
	return lines
}

// priceListPacking describes how a line is sold, e.g. "10 Pairs Per Carton", from the largest visible non-base unit in its unit group.
func priceListPacking(group *domain.ProductLineUnitGroup) string {
	if group == nil || group.BaseUnit == nil {
		return ""
	}
	best := decimal.NewFromInt(1)
	bestName := ""
	for _, associated := range group.AssociatedUnits {
		unit := associated.Unit
		if !associated.IsVisible || unit.IsBaseUnit {
			continue
		}
		ratio := priceListUnitRatio(unit.RatioNumerator, unit.RatioDenominator)
		if ratio.GreaterThan(best) {
			best = ratio
			bestName = unit.Name
		}
	}
	if bestName == "" {
		return ""
	}
	return fmt.Sprintf("%s %s Per %s", best.String(), pluralizeUnit(group.BaseUnit.Name), bestName)
}

func priceListUnitRatio(numerator, denominator string) decimal.Decimal {
	n, err := decimal.NewFromString(numerator)
	if err != nil {
		return decimal.NewFromInt(1)
	}
	d, err := decimal.NewFromString(denominator)
	if err != nil || d.IsZero() {
		return decimal.NewFromInt(1)
	}
	return n.Div(d)
}

func pluralizeUnit(word string) string {
	if word == "" || word[len(word)-1] == 's' {
		return word
	}
	return word + "s"
}
