package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/safeconv"
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
	identity, ok := appctx.Identity[*types.Identity](ctx)
	if !ok || identity == nil {
		return nil, apierror.NewInvariantViolationError("Identity not found in context.")
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainDiscounts, types.ActionRead); apiErr != nil {
		return nil, apiErr
	}
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
			doc.LogoImageType, doc.LogoImage = s.branding.LogoImage(ctx, *account.Branding.LogoURL)
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

	catalog, apiErr := s.loadPriceListCatalog(ctx, accountID, products)
	if apiErr != nil {
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
		OrderedUnitIDs: catalog.priceUnitIDs(products),
	})
	if apiErr != nil {
		return doc, 0, apiErr
	}
	if apiErr := catalog.loadUnits(ctx, s.repos.NewUnitRepo(), accountID, priceListTierUnitIDs(bundle)); apiErr != nil {
		return doc, 0, apiErr
	}

	tiers, dropped := priceListTiersFromBundle(bundle, catalog)
	if dropped > 0 {
		doc.Notes = append(doc.Notes, fmt.Sprintf("%d additional volume breaks are not shown.", dropped))
	}

	doc.Lines = assemblePriceListLines(products, priceListPrices(bundle, products, tiers, catalog), propertyNames, tiers, catalog)
	return doc, safeconv.IntToInt32(len(products)), nil
}

// priceListSellableProducts keeps the products the document can place: those on a product line, which heads their
// section.
func priceListSellableProducts(products []*domain.ProductFull) []*domain.ProductFull {
	out := make([]*domain.ProductFull, 0, len(products))
	for _, product := range products {
		if product.ProductLine != nil && product.Item != nil {
			out = append(out, product)
		}
	}
	return out
}

// priceListCatalog is the units the document quotes in: each item's stocking and order units, and every unit's name
// and ratio.
type priceListCatalog struct {
	itemUnits map[string]*domain.ItemUnits
	units     map[string]*domain.LightUnit
}

// loadPriceListCatalog reads the products' item units and the units they name, once for the whole catalog.
func (s *accountPriceSvcImpl) loadPriceListCatalog(ctx context.Context, accountID string, products []*domain.ProductFull) (*priceListCatalog, *apierror.APIError) {
	itemIDs := make([]string, 0, len(products))
	for _, product := range products {
		itemIDs = append(itemIDs, product.ItemID)
	}
	itemUnits, apiErr := s.repos.NewItemUnitsRepo().Get(ctx, uniqueIDs(itemIDs))
	if apiErr != nil {
		return nil, apiErr
	}
	catalog := &priceListCatalog{itemUnits: itemUnits, units: map[string]*domain.LightUnit{}}
	var unitIDs []string
	for _, u := range itemUnits {
		unitIDs = append(unitIDs, u.AllowedUnitIDs()...)
	}
	return catalog, catalog.loadUnits(ctx, s.repos.NewUnitRepo(), accountID, unitIDs)
}

// loadUnits adds the named units not loaded yet.
func (c *priceListCatalog) loadUnits(ctx context.Context, repo domain.UnitRepo, accountID string, ids []string) *apierror.APIError {
	var missing []string
	for _, id := range uniqueIDs(ids) {
		if _, ok := c.units[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	units, apiErr := repo.GetByIDs(ctx, accountID, missing)
	if apiErr != nil {
		return apiErr
	}
	for _, u := range units {
		c.units[u.ID] = &domain.LightUnit{
			ID: u.ID, Name: u.Name, Abbreviation: u.Abbreviation, RatioNumerator: u.RatioNumerator,
			RatioDenominator: u.RatioDenominator, IsBaseUnit: u.IsBaseUnit,
		}
	}
	return nil
}

// priceUnitID is the unit a product is quoted per: its item's default order unit, else its stocking unit.
func (c *priceListCatalog) priceUnitID(product *domain.ProductFull) string {
	u := c.itemUnits[product.ItemID]
	if u == nil {
		return ""
	}
	if u.DefaultOrderUnitID != "" {
		return u.DefaultOrderUnitID
	}
	return u.StockingUnitID
}

// priceUnitIDs is every unit the catalog is quoted per, which is what the bundle needs to convert into.
func (c *priceListCatalog) priceUnitIDs(products []*domain.ProductFull) []string {
	ids := make([]string, 0, len(products))
	for _, product := range products {
		if id := c.priceUnitID(product); id != "" {
			ids = append(ids, id)
		}
	}
	return uniqueIDs(ids)
}

// measure is how much of the unit's dimension one of unitID holds for this item: the order unit's own quantity of
// stocking units when it sets one, else the unit's ratio.
func (c *priceListCatalog) measure(units *domain.ItemUnits, unitID string) decimal.Decimal {
	if ou := units.OrderUnit(unitID); ou != nil && ou.StockingQuantity != nil {
		if stocking := c.units[units.StockingUnitID]; stocking != nil {
			qty, err := decimal.NewFromString(*ou.StockingQuantity)
			if err == nil {
				return qty.Mul(priceListUnitRatio(stocking.RatioNumerator, stocking.RatioDenominator))
			}
		}
	}
	if u := c.units[unitID]; u != nil {
		return priceListUnitRatio(u.RatioNumerator, u.RatioDenominator)
	}
	return decimal.NewFromInt(1)
}

// packing describes how a product is sold, e.g. "10 Pairs Per Carton": the largest order unit customers can see,
// counted in the unit the product is quoted per. A product with no larger visible unit names its quote unit alone.
func (c *priceListCatalog) packing(product *domain.ProductFull) string {
	units := c.itemUnits[product.ItemID]
	priceUnit := c.units[c.priceUnitID(product)]
	if units == nil || priceUnit == nil {
		return ""
	}
	per := c.measure(units, priceUnit.ID)
	best := decimal.NewFromInt(1)
	bestName := ""
	for _, ou := range units.OrderUnits {
		u := c.units[ou.UnitID]
		if u == nil || ou.UnitID == priceUnit.ID || ou.PortalVisibility == constants.CustomerPortalVisibilityHidden || per.IsZero() {
			continue
		}
		ratio := c.measure(units, ou.UnitID).Div(per)
		if ratio.GreaterThan(best) {
			best, bestName = ratio, u.Name
		}
	}
	if bestName == "" {
		return priceUnit.Name
	}
	return fmt.Sprintf("%s %s Per %s", best.String(), pluralizeUnit(priceUnit.Name), bestName)
}

func priceListProductIDs(products []*domain.ProductFull) []string {
	ids := make([]string, 0, len(products))
	for _, product := range products {
		ids = append(ids, product.ID)
	}
	return ids
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

// priceListTierUnitIDs is every unit a tier column can be counted in.
func priceListTierUnitIDs(bundle *domain.PricingBundle) []string {
	var ids []string
	for _, rule := range bundle.Rules {
		if rule.Price.Type == constants.PriceListPriceTypeTiered {
			ids = append(ids, rule.Price.UnitID)
		}
	}
	for _, discount := range bundle.VolumeDiscounts {
		if len(discount.AcceptableUnitIDs) > 0 {
			ids = append(ids, discount.AcceptableUnitIDs[0])
		}
	}
	return ids
}

// priceListTiersFromBundle builds the candidate price columns: quantity 1 of each product's own quote unit, then each
// distinct break above 1 the customer can reach, from tiered price-list rules (or, before the account's prices are
// mirrored, volume discounts). The bundle already holds only what applies to this buyer.
func priceListTiersFromBundle(bundle *domain.PricingBundle, catalog *priceListCatalog) ([]priceListTier, int) {
	tiers := []priceListTier{priceListTierFor("1", "", nil)}

	type candidate struct {
		quantity decimal.Decimal
		unitID   string
	}
	seen := make(map[string]struct{})
	candidates := make([]candidate, 0)
	add := func(minQuantity, unitID string) {
		threshold, err := decimal.NewFromString(minQuantity)
		if err != nil || unitID == "" || threshold.LessThanOrEqual(decimal.NewFromInt(1)) {
			return
		}
		key := threshold.String() + "\x00" + unitID
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate{quantity: threshold, unitID: unitID})
	}
	for _, rule := range bundle.Rules {
		if rule.Price.Type != constants.PriceListPriceTypeTiered {
			continue
		}
		for _, tier := range rule.Price.Tiers {
			add(tier.MinQuantity, rule.Price.UnitID)
		}
	}
	for _, discount := range bundle.VolumeDiscounts {
		if len(discount.AcceptableUnitIDs) == 0 {
			continue
		}
		for _, tier := range discount.Tiers {
			add(tier.Threshold, discount.AcceptableUnitIDs[0])
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].quantity.LessThan(candidates[j].quantity) })

	dropped := 0
	if len(candidates) > maxPriceListTierColumns-1 {
		dropped = len(candidates) - (maxPriceListTierColumns - 1)
		candidates = candidates[:maxPriceListTierColumns-1]
	}
	for _, c := range candidates {
		tiers = append(tiers, priceListTierFor(c.quantity.String(), c.unitID, catalog.units[c.unitID]))
	}
	return tiers, dropped
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
func priceListPrices(bundle *domain.PricingBundle, products []*domain.ProductFull, tiers []priceListTier, catalog *priceListCatalog) map[string][]string {
	priced := make(map[string][]string, len(products))
	for _, product := range products {
		row := make([]string, len(tiers))
		for t, tier := range tiers {
			unitID := tier.UnitID
			if unitID == "" {
				unitID = catalog.priceUnitID(product)
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
	catalog *priceListCatalog,
) []priceListLine {
	type bucket struct {
		name         string
		baseUnitName string
		products     []priceListProduct
	}
	buckets := make(map[string]*bucket)
	order := make([]string, 0)

	for _, product := range products {
		line := product.ProductLine
		b, ok := buckets[line.ID]
		if !ok {
			b = &bucket{name: line.Name}
			if u := catalog.units[catalog.priceUnitID(product)]; u != nil {
				b.baseUnitName = u.Name
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
			Packing:         catalog.packing(product),
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

func priceListUnitRatio(numerator, denominator string) decimal.Decimal {
	n, err := decimal.NewFromString(numerator)
	if err != nil || n.IsZero() {
		return decimal.NewFromInt(1)
	}
	d, err := decimal.NewFromString(denominator)
	if err != nil || d.IsZero() {
		return decimal.NewFromInt(1)
	}
	return n.Div(d)
}

// A unit name that already carries its own qualifier — "Carton (10 pr)" — reads worse with an "s" welded onto the bracket, so those are left alone.
func pluralizeUnit(word string) string {
	if word == "" || word[len(word)-1] == 's' || word[len(word)-1] == ')' {
		return word
	}
	return word + "s"
}
