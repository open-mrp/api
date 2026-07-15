package recordsep

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
)

// ProductTypeSale is the product type code for physical sale line items — the
// only lines that appear in the pack list's back-order table (charge and
// adjustment lines such as shipping, tax, and credit are excluded).
const productTypeSale = "sale"

type RecordsSvc interface {
	GenPackList(ctx context.Context, req *GenPackListRequest) (*apiresource.PackList, *apierror.APIError)
}

type RecordsSvcConfig struct {
	// ShippingClient (required) is the core-service shipping gRPC client.
	ShippingClient pb.CoreShippingServiceClient
	// SalesClient (required) is the core-service sales gRPC client.
	SalesClient pb.CoreSalesServiceClient
	// AccountClient (required) is the core-service account gRPC client, used for the account name and presigned logo.
	AccountClient pb.CoreServiceClient
}

func (c *RecordsSvcConfig) validate() error {
	if c.ShippingClient == nil {
		return fmt.Errorf("records endpoint service: shipping client is required")
	}
	if c.SalesClient == nil {
		return fmt.Errorf("records endpoint service: sales client is required")
	}
	if c.AccountClient == nil {
		return fmt.Errorf("records endpoint service: account client is required")
	}
	return nil
}

type recordsSvcImpl struct {
	shippingClient pb.CoreShippingServiceClient
	salesClient    pb.CoreSalesServiceClient
	accountClient  pb.CoreServiceClient
}

var recordsSvcTracer = tracing.GetTracer("api-gateway.endpoints.records.service")

func NewRecordsSvc(config *RecordsSvcConfig) RecordsSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &recordsSvcImpl{
		shippingClient: config.ShippingClient,
		salesClient:    config.SalesClient,
		accountClient:  config.AccountClient,
	}
}

// GenPackList assembles a pack-list document for a shipment by composing the
// shipment (with its lines and shipping cases), its parent sales order (with
// lines and email contacts), and the selling account's name and presigned logo.
func (m *recordsSvcImpl) GenPackList(ctx context.Context, req *GenPackListRequest) (*apiresource.PackList, *apierror.APIError) {
	shipResp, apiErr := grpcutil.CallRPC(ctx, recordsSvcTracer, "service.records.gen_pack_list.get_shipment", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShipmentResponse, error) {
			return m.shippingClient.GetShipment(ctx, &pb.GetShipmentRequest{
				Id:       req.ShipmentID,
				Includes: []string{"lines", "shipping_cases"},
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if shipResp == nil || shipResp.Shipment == nil {
		return nil, apierror.NewResourceNotFoundError("Shipment not found.")
	}
	s := shipResp.Shipment

	orderResp, apiErr := grpcutil.CallRPC(ctx, recordsSvcTracer, "service.records.gen_pack_list.get_sales_order", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetSalesOrderResponse, error) {
			return m.salesClient.GetSalesOrder(ctx, &pb.GetSalesOrderRequest{
				Id:       s.SalesOrderId,
				Includes: []string{"lines", "contacts"},
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if orderResp == nil || orderResp.SalesOrder == nil {
		return nil, apierror.NewResourceNotFoundError("Sales order not found.")
	}
	o := orderResp.SalesOrder

	accountName, logoURL := m.loadAccountHeader(ctx, s.AccountId)

	return assemblePackList(s, o, accountName, logoURL), nil
}

// loadAccountHeader resolves the selling account's display name and a presigned
// logo URL. The logo is best-effort: a missing logo or presign failure yields a
// nil URL rather than failing the whole document, mirroring the legacy behavior.
func (m *recordsSvcImpl) loadAccountHeader(ctx context.Context, accountID string) (string, *string) {
	name := ""
	acctResp, apiErr := grpcutil.CallRPC(ctx, recordsSvcTracer, "service.records.gen_pack_list.get_account", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountResponse, error) {
			return m.accountClient.GetAccount(ctx, &pb.GetAccountRequest{Id: accountID}, opts...)
		})
	if apiErr == nil && acctResp != nil && acctResp.Account != nil {
		name = acctResp.Account.Name
	}

	var logoURL *string
	logoResp, apiErr := grpcutil.CallRPC(ctx, recordsSvcTracer, "service.records.gen_pack_list.get_logo", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountLogoURLResponse, error) {
			return m.accountClient.GetAccountLogoURL(ctx, &pb.GetAccountLogoURLRequest{Id: accountID}, opts...)
		})
	if apiErr == nil && logoResp != nil && logoResp.GetUrl() != "" {
		url := logoResp.GetUrl()
		logoURL = &url
	}

	return name, logoURL
}

// assemblePackList maps a shipment + its sales order into the pack-list document.
func assemblePackList(s *pb.ShipmentInfo, o *pb.SalesOrderInfo, accountName string, logoURL *string) *apiresource.PackList {
	// Line numbers live on the sales-order line, not the shipment line; index them
	// by sales-order-line id so each packed shipment line can carry its number.
	lineNumberByOrderLineID := make(map[string]int32, len(o.Lines))
	for _, l := range o.Lines {
		lineNumberByOrderLineID[l.Id] = l.LineItemNumber
	}

	pl := &apiresource.PackList{
		Object:             constants.ObjectTypePackList,
		AccountName:        accountName,
		AccountLogoURL:     logoURL,
		SalesOrderNumber:   o.GetNumber(),
		CustomerPO:         strPtrOrNil(o.GetCustomerPoNumber()),
		ShipmentNumber:     s.GetNumber(),
		ShippedAt:          grpcutil.TimestampToTimePtr(s.ShippedAt),
		BillTo:             billToParty(o),
		ShipTo:             shipToParty(o),
		ContactInformation: contactInformation(o),
		Carrier:            strPtrOrNil(s.GetCarrierName()),
		CarrierOption:      strPtrOrNil(s.GetServiceLevelName()),
		Priority:           strPtrOrNil(o.GetPriorityName()),
		PaymentTerm:        strPtrOrNil(o.GetPaymentTermName()),
		SalesRep:           strPtrOrNil(o.GetSalesRepName()),
		ShippingCases:      apiresource.NewList(packListCases(s), apiresource.PageInfo{}),
		LineItems:          apiresource.NewList(packListLineItems(s, lineNumberByOrderLineID), apiresource.PageInfo{}),
		BackOrders:         apiresource.NewList(packListBackOrders(o), apiresource.PageInfo{}),
	}
	return pl
}

func billToParty(o *pb.SalesOrderInfo) *apiresource.PackListParty {
	return &apiresource.PackListParty{
		Object:      constants.ObjectTypePackListParty,
		Name:        o.GetBillToName(),
		StreetLine1: strPtrOrNil(o.GetBillToStreetLine_1()),
		StreetLine2: strPtrOrNil(o.GetBillToStreetLine_2()),
		Locality:    strPtrOrNil(o.GetBillToLocality()),
		State:       strPtrOrNil(o.GetBillToState()),
		PostalCode:  strPtrOrNil(o.GetBillToPostalCode()),
		Country:     strPtrOrNil(o.GetBillToCountry()),
		Phone:       strPtrOrNil(o.GetBillToPhone()),
		Email:       strPtrOrNil(o.GetBillToEmail()),
	}
}

func shipToParty(o *pb.SalesOrderInfo) *apiresource.PackListParty {
	return &apiresource.PackListParty{
		Object:      constants.ObjectTypePackListParty,
		Name:        o.GetShipToName(),
		StreetLine1: strPtrOrNil(o.GetShipToStreetLine_1()),
		StreetLine2: strPtrOrNil(o.GetShipToStreetLine_2()),
		Locality:    strPtrOrNil(o.GetShipToLocality()),
		State:       strPtrOrNil(o.GetShipToState()),
		PostalCode:  strPtrOrNil(o.GetShipToPostalCode()),
		Country:     strPtrOrNil(o.GetShipToCountry()),
		Phone:       strPtrOrNil(o.GetShipToPhone()),
		Email:       strPtrOrNil(o.GetShipToEmail()),
	}
}

// contactInformation collects the document's contact lines: the order's email
// recipients (lowercased) followed by the billing contact phone, if any.
func contactInformation(o *pb.SalesOrderInfo) []string {
	emails := o.GetInvoiceEmails()
	info := make([]string, 0, len(emails)+1)
	for _, email := range emails {
		if email != "" {
			info = append(info, strings.ToLower(email))
		}
	}
	if phone := o.GetBillToPhone(); phone != "" {
		info = append(info, phone)
	}
	return info
}

// packListCases maps the shipment's shipping cases, sorted by case number.
func packListCases(s *pb.ShipmentInfo) []apiresource.PackListCase {
	cases := make([]apiresource.PackListCase, 0, len(s.ShippingCases))
	for _, c := range s.ShippingCases {
		cases = append(cases, apiresource.PackListCase{
			Object:         constants.ObjectTypePackListCase,
			Number:         c.GetNumber(),
			Weight:         c.GetFreightWeightValue(),
			WeightUnit:     c.GetFreightWeightUnitAbbreviation(),
			TrackingNumber: strPtrOrNil(c.GetTrackingNumber()),
			Carrier:        strPtrOrNil(c.GetCarrierName()),
		})
	}
	sort.SliceStable(cases, func(i, j int) bool { return cases[i].Number < cases[j].Number })
	return cases
}

// packListLineItems maps the shipment's packed lines, carrying each line's
// number from its sales-order line, sorted by line number.
func packListLineItems(s *pb.ShipmentInfo, lineNumberByOrderLineID map[string]int32) []apiresource.PackListLineItem {
	items := make([]apiresource.PackListLineItem, 0, len(s.Lines))
	for _, l := range s.Lines {
		var lineNumber *int32
		if n, ok := lineNumberByOrderLineID[l.GetSalesOrderLineId()]; ok {
			n := n
			lineNumber = &n
		}
		items = append(items, apiresource.PackListLineItem{
			Object:         constants.ObjectTypePackListLineItem,
			LineItemNumber: lineNumber,
			SKU:            l.GetOrderLineSku(),
			Description:    l.GetOrderLineDescription(),
			Quantity:       l.GetQuantityValue(),
			Unit:           l.GetQuantityUnitName(),
		})
	}
	sortByLineNumber(items, func(i int) *int32 { return items[i].LineItemNumber })
	return items
}

// packListBackOrders maps the order's sale-type lines that still have quantity
// ordered beyond what has been packed, sorted by line number.
func packListBackOrders(o *pb.SalesOrderInfo) []apiresource.PackListBackOrder {
	rows := make([]apiresource.PackListBackOrder, 0)
	for _, l := range o.Lines {
		if l.GetProductTypeCode() != productTypeSale {
			continue
		}
		ordered := parseDecimal(l.GetQuantityValue())
		packed := parseDecimal(l.GetQuantityPackedValue())
		backOrdered := ordered.Sub(packed)
		if backOrdered.LessThanOrEqual(decimal.Zero) {
			continue
		}
		lineNumber := l.GetLineItemNumber()
		rows = append(rows, apiresource.PackListBackOrder{
			Object:              constants.ObjectTypePackListBackOrder,
			LineItemNumber:      &lineNumber,
			SKU:                 l.GetProductSku(),
			Description:         l.GetProductDescription(),
			QuantityOrdered:     ordered.String(),
			QuantityShipped:     packed.String(),
			QuantityBackOrdered: backOrdered.String(),
			Unit:                l.GetQuantityUnitName(),
		})
	}
	sortByLineNumber(rows, func(i int) *int32 { return rows[i].LineItemNumber })
	return rows
}

// sortByLineNumber sorts a slice in place by an optional line number, placing
// entries without a number last.
func sortByLineNumber[T any](items []T, numberOf func(i int) *int32) {
	sort.SliceStable(items, func(i, j int) bool {
		ni, nj := numberOf(i), numberOf(j)
		if ni == nil {
			return false
		}
		if nj == nil {
			return true
		}
		return *ni < *nj
	})
}

func parseDecimal(s string) decimal.Decimal {
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
