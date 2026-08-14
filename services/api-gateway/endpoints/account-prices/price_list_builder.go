package accountpriceep

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"  // register GIF decoder for image.Decode
	_ "image/jpeg" // register JPEG decoder for image.Decode
	"image/png"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	_ "golang.org/x/image/webp" // register WEBP decoder (account logos are often .webp)
	"google.golang.org/grpc"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
)

const (
	// productPageSize is the page size used to walk the customer's catalog.
	productPageSize = 500
	// maxPricedProducts caps the export. A price list beyond this is not a document anyone reads, and every extra product multiplies the quote calls below.
	maxPricedProducts = 5000
	// maxTierColumns bounds how many volume breaks become columns. Anything past this is dropped and disclosed on the title page rather than silently truncated.
	maxTierColumns = 4
	// quoteBatchSize keeps each price-quote RPC well inside the request deadline.
	quoteBatchSize = 250
)

// priceListInputs is everything gathered before layout: the catalog, the header facts, and any disclosure the document has to carry.
type priceListInputs struct {
	document priceListDocument
	notes    []string
}

// buildCustomerPriceList assembles the priced, grouped catalog for one customer.
//
// Prices come from the same engine that prices a sales order — every figure here is produced by QuoteSalesOrderLinePrices, so the document cannot drift from what the customer is actually charged. Volume breaks are discovered by observation: the catalog is quoted at each candidate quantity and a tier only becomes a column when it actually moves a price, which avoids reimplementing discount scoping here.
func (m *accountPriceSvcImpl) buildCustomerPriceList(ctx context.Context, customerID string, asOf time.Time) (*priceListDocument, *apierror.APIError) {
	inputs := &priceListInputs{}

	customer, apiErr := m.fetchPriceListCustomer(ctx, customerID)
	if apiErr != nil {
		return nil, apiErr
	}
	inputs.document.CustomerName = customer.Name
	inputs.document.DateLong = asOf.Format("January 2, 2006")
	if term := customer.GetDefaultPaymentTerm(); term != nil {
		inputs.document.PaymentTerm = term.Name
	}
	if term := customer.GetDefaultShippingTerm(); term != nil {
		inputs.document.ShippingTerm = term.Name
	}

	m.fillPriceListMerchant(ctx, &inputs.document)

	products, truncated, apiErr := m.fetchPriceListProducts(ctx, customerID)
	if apiErr != nil {
		return nil, apiErr
	}
	if truncated {
		inputs.notes = append(inputs.notes, fmt.Sprintf("Showing the first %d products; this customer has access to more.", maxPricedProducts))
	}
	if len(products) == 0 {
		inputs.notes = append(inputs.notes, "No products are currently available to this customer.")
		inputs.document.Notes = inputs.notes
		return &inputs.document, nil
	}

	propertyNames, apiErr := m.fetchPriceListPropertyNames(ctx, products)
	if apiErr != nil {
		return nil, apiErr
	}

	tiers, tiersDropped, apiErr := m.fetchPriceListTiers(ctx, customerID, products)
	if apiErr != nil {
		return nil, apiErr
	}
	if tiersDropped > 0 {
		inputs.notes = append(inputs.notes, fmt.Sprintf("%d additional volume breaks are not shown.", tiersDropped))
	}

	priced, apiErr := m.quotePriceListProducts(ctx, customerID, products, tiers)
	if apiErr != nil {
		return nil, apiErr
	}

	inputs.document.Lines = assemblePriceListLines(products, priced, propertyNames, tiers)
	inputs.document.Notes = inputs.notes
	return &inputs.document, nil
}

func (m *accountPriceSvcImpl) fetchPriceListCustomer(ctx context.Context, customerID string) (*pb.CustomerProto, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.price_list.customer", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCustomerResponse, error) {
			return m.coreClient.GetCustomer(ctx, &pb.GetCustomerRequest{Id: customerID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if resp.GetCustomer() == nil {
		return nil, apierror.NewResourceNotFoundError("Customer not found.")
	}
	return resp.GetCustomer(), nil
}

// fillPriceListMerchant adds the merchant's own name and logo to the title page. Best-effort: a price list without letterhead is still a usable price list.
func (m *accountPriceSvcImpl) fillPriceListMerchant(ctx context.Context, doc *priceListDocument) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || identity.Target == nil || identity.Target.AccountID == "" {
		return
	}
	accountID := identity.Target.AccountID
	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.price_list.account", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountResponse, error) {
			return m.coreClient.GetAccount(ctx, &pb.GetAccountRequest{Id: accountID}, opts...)
		})
	if apiErr != nil || resp.GetAccount() == nil {
		return
	}
	doc.MerchantName = resp.GetAccount().GetName()
	if branding := resp.GetAccount().GetBranding(); branding != nil {
		doc.LogoImageType, doc.LogoImage = fetchPriceListLogo(ctx, branding.GetLogoUrl())
	}
}

// fetchPriceListProducts walks every product the customer may buy. The customer_ids filter applies the same three visibility pathways order entry uses (direct relation, account group, price group), so the document can never quote something they cannot order.
func (m *accountPriceSvcImpl) fetchPriceListProducts(ctx context.Context, customerID string) ([]*pb.ProductFullInfo, bool, *apierror.APIError) {
	products := make([]*pb.ProductFullInfo, 0, productPageSize)
	var cursor *string

	for {
		req := &pb.ListProductsFullRequest{
			Limit:       productPageSize,
			CustomerIds: []string{customerID},
			Cursor:      cursor,
		}
		resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.price_list.products", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductsFullResponse, error) {
				return m.coreClient.ListProductsFull(ctx, req, opts...)
			})
		if apiErr != nil {
			return nil, false, apiErr
		}

		for _, product := range resp.GetProducts() {
			// A product with no line can never match an account price or a line-scoped discount, and has no unit group to price against.
			if product.GetProductLine() == nil {
				continue
			}
			products = append(products, product)
			if len(products) >= maxPricedProducts {
				return products, true, nil
			}
		}

		page := resp.GetPageInfo()
		if page == nil || !page.GetHasNextPage() || page.GetNextCursor() == "" {
			break
		}
		next := page.GetNextCursor()
		cursor = &next
	}
	return products, false, nil
}

// fetchPriceListPropertyNames resolves the property behind each attribute, since the product payload carries only property_id and the table needs column headings.
func (m *accountPriceSvcImpl) fetchPriceListPropertyNames(ctx context.Context, products []*pb.ProductFullInfo) (map[string]string, *apierror.APIError) {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, product := range products {
		for _, attribute := range product.GetItem().GetAttributes() {
			id := attribute.GetPropertyId()
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.price_list.properties", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPropertiesByIDsResponse, error) {
			return m.coreClient.BatchGetPropertiesByIDs(ctx, &pb.BatchGetPropertiesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	names := make(map[string]string, len(ids))
	for _, property := range resp.GetProperties() {
		names[property.GetId()] = property.GetName()
	}
	return names, nil
}

// fetchPriceListTiers builds the candidate price columns: quantity 1, then each distinct volume-discount threshold reachable by this customer. Which of them survive is decided later, by whether they actually change a price.
func (m *accountPriceSvcImpl) fetchPriceListTiers(ctx context.Context, customerID string, products []*pb.ProductFullInfo) ([]priceListTier, int, *apierror.APIError) {
	baseUnitID, baseUnitAbbr := priceListBaseUnit(products)
	tiers := []priceListTier{{Label: "1+", Quantity: "1", UnitID: baseUnitID, UnitAbbreviation: baseUnitAbbr}}

	resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.price_list.discounts", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListVolumeDiscountsResponse, error) {
			return m.salesClient.ListVolumeDiscounts(ctx, &pb.ListVolumeDiscountsRequest{
				Limit:             200,
				CustomerAccountId: &customerID,
				Includes:          []string{"tiers", "acceptable_units"},
			}, opts...)
		})
	if apiErr != nil {
		// A price list without volume columns is still correct; it just shows the single-unit price. Do not fail the whole export over the discount lookup.
		return tiers, 0, nil
	}

	type candidate struct {
		quantity decimal.Decimal
		unitID   string
		unitAbbr string
	}
	seen := make(map[string]struct{})
	candidates := make([]candidate, 0)
	for _, discount := range resp.GetVolumeDiscounts() {
		units := discount.GetAcceptableUnits()
		if len(units) == 0 {
			continue
		}
		unit := units[0]
		for _, tier := range discount.GetTiers() {
			threshold, err := decimal.NewFromString(tier.GetThreshold())
			if err != nil || threshold.LessThanOrEqual(decimal.NewFromInt(1)) {
				continue
			}
			key := threshold.String() + "\x00" + unit.GetId()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, candidate{quantity: threshold, unitID: unit.GetId(), unitAbbr: unit.GetAbbreviation()})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].quantity.LessThan(candidates[j].quantity)
	})

	dropped := 0
	if len(candidates) > maxTierColumns-1 {
		dropped = len(candidates) - (maxTierColumns - 1)
		candidates = candidates[:maxTierColumns-1]
	}
	for _, c := range candidates {
		tiers = append(tiers, priceListTier{
			Label:            c.quantity.String() + "+ " + c.unitAbbr,
			Quantity:         c.quantity.String(),
			UnitID:           c.unitID,
			UnitAbbreviation: c.unitAbbr,
		})
	}
	return tiers, dropped, nil
}

// quotePriceListProducts prices every product at every candidate tier, returning prices[productID][tierIndex]. Each tier is a separate pass because the engine sums quantities across the lines in one request, so mixing quantities would distort them.
func (m *accountPriceSvcImpl) quotePriceListProducts(
	ctx context.Context,
	customerID string,
	products []*pb.ProductFullInfo,
	tiers []priceListTier,
) (map[string][]string, *apierror.APIError) {
	priced := make(map[string][]string, len(products))
	for _, product := range products {
		priced[product.GetId()] = make([]string, len(tiers))
	}

	for t, tier := range tiers {
		for start := 0; start < len(products); start += quoteBatchSize {
			end := min(start+quoteBatchSize, len(products))

			lines := make([]*pb.QuoteSalesOrderLineInput, 0, end-start)
			for _, product := range products[start:end] {
				unitID := tier.UnitID
				if unitID == "" {
					unitID = priceListProductBaseUnitID(product)
				}
				lines = append(lines, &pb.QuoteSalesOrderLineInput{
					ProductId:      product.GetId(),
					QuantityValue:  tier.Quantity,
					QuantityUnitId: unitID,
				})
			}

			resp, apiErr := grpcutil.CallRPC(ctx, accountPriceSvcTracer, "service.account_prices.price_list.quote", domain.ServiceName,
				func(ctx context.Context, opts ...grpc.CallOption) (*pb.QuoteSalesOrderLinePricesResponse, error) {
					return m.salesClient.QuoteSalesOrderLinePrices(ctx, &pb.QuoteSalesOrderLinePricesRequest{
						BuyerAccountId: customerID,
						Lines:          lines,
					}, opts...)
				})
			if apiErr != nil {
				return nil, apiErr
			}

			for _, quote := range resp.GetLines() {
				row, ok := priced[quote.GetProductId()]
				if !ok {
					continue
				}
				row[t] = formatPriceListMoney(quote.GetUnitPriceValue())
			}
		}
	}
	return priced, nil
}

// assemblePriceListLines turns the priced catalog into per-product-line documents.
func assemblePriceListLines(
	products []*pb.ProductFullInfo,
	priced map[string][]string,
	propertyNames map[string]string,
	tiers []priceListTier,
) []priceListLine {
	type lineBucket struct {
		name         string
		baseUnitName string
		packing      string
		products     []priceListProduct
	}
	buckets := make(map[string]*lineBucket)
	order := make([]string, 0)

	for _, product := range products {
		line := product.GetProductLine()
		bucket, ok := buckets[line.GetId()]
		if !ok {
			bucket = &lineBucket{
				name:         line.GetName(),
				baseUnitName: line.GetUnitGroup().GetBaseUnit().GetName(),
				packing:      priceListPacking(line.GetUnitGroup()),
			}
			buckets[line.GetId()] = bucket
			order = append(order, line.GetId())
		}

		values := make(map[string]string)
		orders := make(map[string]int32)
		for _, attribute := range product.GetItem().GetAttributes() {
			name := propertyNames[attribute.GetPropertyId()]
			if name == "" {
				continue
			}
			values[name] = attribute.GetValue()
			orders[name] = attribute.GetSortOrder()
		}

		bucket.products = append(bucket.products, priceListProduct{
			ProductID:       product.GetId(),
			SKU:             product.GetItem().GetSku(),
			Description:     product.GetItem().GetDescription(),
			AttributeValues: values,
			AttributeOrders: orders,
			Prices:          priced[product.GetId()],
			Packing:         bucket.packing,
		})
	}

	sort.SliceStable(order, func(i, j int) bool {
		return buckets[order[i]].name < buckets[order[j]].name
	})

	lines := make([]priceListLine, 0, len(order))
	for _, id := range order {
		bucket := buckets[id]
		lines = append(lines, priceListLine{
			ProductLineID:   id,
			ProductLineName: bucket.name,
			BaseUnitName:    bucket.baseUnitName,
			Sections:        buildPriceListSections(bucket.products, tiers),
		})
	}
	return lines
}

// priceListPacking describes how the line is sold, e.g. "10 Pairs Per Carton", from the largest visible non-base unit in its unit group.
func priceListPacking(group *pb.ItemCategoryUnitGroupInfo) string {
	if group == nil {
		return ""
	}
	baseName := group.GetBaseUnit().GetName()
	best := decimal.NewFromInt(1)
	bestName := ""
	for _, associated := range group.GetAssociatedUnits() {
		unit := associated.GetUnit()
		if unit == nil || !associated.GetIsVisible() || unit.GetIsBaseUnit() {
			continue
		}
		ratio := priceListRatio(unit)
		if ratio.GreaterThan(best) {
			best = ratio
			bestName = unit.GetName()
		}
	}
	if bestName == "" || baseName == "" {
		return ""
	}
	return fmt.Sprintf("%s %s Per %s", best.String(), pluralize(baseName), bestName)
}

func priceListRatio(unit *pb.UnitInfo) decimal.Decimal {
	numerator, err := decimal.NewFromString(unit.GetRatioNumerator())
	if err != nil {
		return decimal.NewFromInt(1)
	}
	denominator, err := decimal.NewFromString(unit.GetRatioDenominator())
	if err != nil || denominator.IsZero() {
		return decimal.NewFromInt(1)
	}
	return numerator.Div(denominator)
}

// priceListBaseUnit picks the base unit shared by most of the catalog, used for the quantity-1 column when a tier carries no unit of its own.
func priceListBaseUnit(products []*pb.ProductFullInfo) (string, string) {
	counts := make(map[string]int)
	abbreviations := make(map[string]string)
	for _, product := range products {
		unit := product.GetProductLine().GetUnitGroup().GetBaseUnit()
		if unit == nil {
			continue
		}
		counts[unit.GetId()]++
		abbreviations[unit.GetId()] = unit.GetAbbreviation()
	}
	bestID, bestCount := "", 0
	for id, count := range counts {
		if count > bestCount || (count == bestCount && id < bestID) {
			bestID, bestCount = id, count
		}
	}
	return bestID, abbreviations[bestID]
}

func priceListProductBaseUnitID(product *pb.ProductFullInfo) string {
	return product.GetProductLine().GetUnitGroup().GetBaseUnit().GetId()
}

func formatPriceListMoney(value string) string {
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return ""
	}
	return "$" + amount.StringFixed(2)
}

func pluralize(word string) string {
	if word == "" || strings.HasSuffix(word, "s") {
		return word
	}
	return word + "s"
}

// fetchPriceListLogo downloads the account logo and re-encodes it as PNG. Best-effort: the title page simply omits the logo when anything goes wrong.
func fetchPriceListLogo(ctx context.Context, url string) (string, []byte) {
	if strings.TrimSpace(url) == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil
	}
	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL from account branding logo stored server-side
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil || len(body) == 0 {
		return "", nil
	}
	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return "", nil
	}
	// Flatten transparency onto white: fpdf renders alpha PNGs unreliably and the page behind the logo is white anyway.
	bounds := img.Bounds()
	flat := image.NewRGBA(bounds)
	draw.Draw(flat, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, bounds, img, bounds.Min, draw.Over)
	var out bytes.Buffer
	if err := png.Encode(&out, flat); err != nil {
		return "", nil
	}
	return "PNG", out.Bytes()
}
