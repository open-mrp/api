package searchep

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	agentpb "github.com/augno/api/shared/proto/agent"
	corepb "github.com/augno/api/shared/proto/core"
	notifpb "github.com/augno/api/shared/proto/notification"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

// SearchSvc backs the unified search endpoint by fanning out to the per-type list RPCs and projecting their results into lightweight Entity references.
type SearchSvc interface {
	Search(ctx context.Context, req *SearchRequest) (*apiresource.List[apiresource.Entity], *apierror.APIError)
}

type SearchSvcConfig struct {
	// CoreClient (required) backs the invoice, customer, item, and product searches.
	CoreClient corepb.CoreServiceClient
	// SalesClient (required) backs the sales-order search.
	SalesClient corepb.CoreSalesServiceClient
	// PurchaseClient (required) backs the purchase-order search.
	PurchaseClient corepb.CorePurchaseServiceClient
	// ShippingClient (required) backs the shipment search.
	ShippingClient corepb.CoreShippingServiceClient
	// ChatClient (required) backs the messaging-contact search.
	ChatClient notifpb.ChatServiceClient
	// AgentClient (required) backs the agent search.
	AgentClient agentpb.AgentServiceClient
}

func (c *SearchSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("search endpoint service: core client is required")
	}
	if c.SalesClient == nil {
		return fmt.Errorf("search endpoint service: sales client is required")
	}
	if c.PurchaseClient == nil {
		return fmt.Errorf("search endpoint service: purchase client is required")
	}
	if c.ShippingClient == nil {
		return fmt.Errorf("search endpoint service: shipping client is required")
	}
	if c.ChatClient == nil {
		return fmt.Errorf("search endpoint service: chat client is required")
	}
	if c.AgentClient == nil {
		return fmt.Errorf("search endpoint service: agent client is required")
	}
	return nil
}

type searchSvcImpl struct {
	coreClient     corepb.CoreServiceClient
	salesClient    corepb.CoreSalesServiceClient
	purchaseClient corepb.CorePurchaseServiceClient
	shippingClient corepb.CoreShippingServiceClient
	chatClient     notifpb.ChatServiceClient
	agentClient    agentpb.AgentServiceClient
	providers      []searchProvider
}

// searchScope narrows a search beyond the free-text term. A non-empty customerID restricts results to
// that customer's own records (the customer's account id) — used so a customer-visible reply can only
// tag resources the customer is entitled to see.
type searchScope struct {
	customerID string
}

// searchProvider searches a single resource type. permission is the read permission the caller must hold for results of this type to be included. customerSafe marks a provider whose results may be offered when the search is scoped to a customer (the provider must honor scope.customerID); providers that are not customer-safe are excluded entirely from a customer-scoped search.
type searchProvider struct {
	objectType   constants.ObjectType
	permission   types.Permission
	customerSafe bool
	search       func(ctx context.Context, query string, limit int32, scope searchScope) ([]apiresource.Entity, *apierror.APIError)
}

var searchSvcTracer = tracing.GetTracer("api-gateway.endpoints.search.service")

// searchProviderTimeout bounds each per-type list RPC in the fan-out. Well under the default RPC deadline on purpose: a picker that takes ten seconds to populate has already lost the user, and a type that slow contributes nothing worth waiting for.
const searchProviderTimeout = 5 * time.Second

func NewSearchSvc(config *SearchSvcConfig) SearchSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	svc := &searchSvcImpl{
		coreClient:     config.CoreClient,
		salesClient:    config.SalesClient,
		purchaseClient: config.PurchaseClient,
		shippingClient: config.ShippingClient,
		chatClient:     config.ChatClient,
		agentClient:    config.AgentClient,
	}
	// customerSafe providers (sales orders, invoices, shipments) honor scope.customerID and may be
	// offered when composing a customer-visible reply; every other type is excluded from a
	// customer-scoped search so a tag can never reference a record the customer may not see.
	svc.providers = []searchProvider{
		{constants.ObjectTypeSalesOrder, types.Permission{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}, true, svc.searchSalesOrders},
		{constants.ObjectTypePurchaseOrder, types.Permission{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionRead}, false, svc.searchPurchaseOrders},
		{constants.ObjectTypeInvoice, types.Permission{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}, true, svc.searchInvoices},
		{constants.ObjectTypeCustomer, types.Permission{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}, false, svc.searchCustomers},
		{constants.ObjectTypeItem, types.Permission{Domain: types.PermissionDomainItems, Action: types.ActionRead}, false, svc.searchItems},
		{constants.ObjectTypeProduct, types.Permission{Domain: types.PermissionDomainItems, Action: types.ActionRead}, false, svc.searchProducts},
		{constants.ObjectTypeShipment, types.Permission{Domain: types.PermissionDomainShipments, Action: types.ActionRead}, true, svc.searchShipments},
		{constants.ObjectTypeMessagingContact, types.Permission{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}, false, svc.searchContacts},
		{constants.ObjectTypeAgentDefinition, types.Permission{Domain: types.PermissionDomainAgents, Action: types.ActionRead}, false, svc.searchAgents},
	}
	return svc
}

func (m *searchSvcImpl) Search(ctx context.Context, req *SearchRequest) (*apiresource.List[apiresource.Entity], *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, apierror.NewAuthorizationError("You do not have permission to access this resource.")
	}

	query := searchQueryFromRequest(req)

	typeFilter, apiErr := searchTypeFilterFromRequest(req)
	if apiErr != nil {
		return nil, apiErr
	}

	// An empty query lists recent rows of each type, which is useful for browsing a single scoped category (the chat `@`-picker drills into one type before the user types). Allowing it unscoped would fan a bare "list everything" across every type, so require a query unless the search is narrowed with `types`.
	if query == "" && len(typeFilter) == 0 {
		return nil, apierror.NewParameterMissingError("The q query parameter is required unless the search is scoped with the types parameter.", "q")
	}

	scope := searchScope{}
	if req.Customer != nil {
		scope.customerID = strings.TrimSpace(*req.Customer)
	}

	active := selectActiveSearchProviders(m.providers, typeFilter, identity, scope)

	if len(active) == 0 {
		if apiErr := identity.CheckHasAnyPermission(searchReadPermissions...); apiErr != nil {
			return nil, apiErr
		}
		return apiresource.NewList[apiresource.Entity](nil, apiresource.PageInfo{}), nil
	}

	// Fan out concurrently. A single type failing degrades to no results for that type rather than failing the whole search, so the picker stays usable.
	//
	// Providers are capped well below the default RPC deadline so a slow type degrades the same way one that errors does. Search backs an interactive picker: the slowest of nine concurrent list RPCs sets the latency the user feels, and a type that would need the full deadline is worth dropping rather than waiting for.
	searchCtx, cancelSearch := context.WithTimeout(ctx, searchProviderTimeout)
	defer cancelSearch()

	results := make([][]apiresource.Entity, len(active))
	var wg sync.WaitGroup
	for i, p := range active {
		wg.Add(1)
		go func(i int, p searchProvider) {
			defer wg.Done()
			ents, apiErr := p.search(searchCtx, query, req.Limit, scope)
			if apiErr != nil {
				slog.WarnContext(ctx, "search provider failed", "type", string(p.objectType), "error", apiErr.Error())
				return
			}
			results[i] = ents
		}(i, p)
	}
	wg.Wait()

	return apiresource.NewList(interleave(results, int(req.Limit)), apiresource.PageInfo{}), nil
}

// searchQueryFromRequest normalizes the free-text term, collapsing a missing or whitespace-only value to the empty string. Callers decide whether an empty query is allowed (it is only when the search is scoped with `types`).
func searchQueryFromRequest(req *SearchRequest) string {
	if req.Query == nil {
		return ""
	}
	return strings.TrimSpace(*req.Query)
}

func searchTypeFilterFromRequest(req *SearchRequest) (map[constants.ObjectType]bool, *apierror.APIError) {
	if len(req.Types) == 0 {
		return nil, nil
	}
	typeFilter := make(map[constants.ObjectType]bool, len(req.Types))
	for _, objectType := range req.Types {
		if !isSearchObjectType(objectType) {
			return nil, apierror.NewParameterInvalidError(
				fmt.Sprintf("Field 'types' must be one of the searchable resource types (for example %s or %s).", constants.ObjectTypeSalesOrder, constants.ObjectTypeCustomer),
				"types",
			)
		}
		typeFilter[objectType] = true
	}
	return typeFilter, nil
}

func selectActiveSearchProviders(providers []searchProvider, typeFilter map[constants.ObjectType]bool, identity *types.Identity, scope searchScope) []searchProvider {
	active := make([]searchProvider, 0, len(providers))
	for _, p := range providers {
		if len(typeFilter) > 0 && !typeFilter[p.objectType] {
			continue
		}
		// A customer-scoped search only ever surfaces customer-safe types, so a tag composed into a
		// customer-visible reply can never reference a privileged record.
		if scope.customerID != "" && !p.customerSafe {
			continue
		}
		if identity.CheckHasPermission(p.permission.Domain, p.permission.Action) != nil {
			continue
		}
		active = append(active, p)
	}
	return active
}

// interleave round-robins across per-type result slices so a short result set is not dominated by a single type, capping the total at limit.
func interleave(results [][]apiresource.Entity, limit int) []apiresource.Entity {
	out := make([]apiresource.Entity, 0, limit)
	for col := 0; len(out) < limit; col++ {
		progressed := false
		for _, ents := range results {
			if col >= len(ents) {
				continue
			}
			progressed = true
			out = append(out, ents[col])
			if len(out) >= limit {
				return out
			}
		}
		if !progressed {
			break
		}
	}
	return out
}

func (m *searchSvcImpl) searchSalesOrders(ctx context.Context, query string, limit int32, scope searchScope) ([]apiresource.Entity, *apierror.APIError) {
	req := &corepb.ListSalesOrdersRequest{Query: &query, Limit: limit}
	if scope.customerID != "" {
		req.CustomerIds = []string{scope.customerID}
	}
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.sales_orders", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.ListSalesOrdersResponse, error) {
			return m.salesClient.ListSalesOrders(ctx, req, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]apiresource.Entity, 0, len(resp.SalesOrders))
	for _, o := range resp.SalesOrders {
		name := o.Number
		out = append(out, *apiresource.NewEntity(o.Id, constants.ObjectTypeSalesOrder, &name, nonEmpty(o.CustomerName)))
	}
	return out, nil
}

func (m *searchSvcImpl) searchPurchaseOrders(ctx context.Context, query string, limit int32, _ searchScope) ([]apiresource.Entity, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.purchase_orders", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.ListPurchaseOrdersResponse, error) {
			return m.purchaseClient.ListPurchaseOrders(ctx, &corepb.ListPurchaseOrdersRequest{Query: &query, Limit: limit}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]apiresource.Entity, 0, len(resp.PurchaseOrders))
	for _, o := range resp.PurchaseOrders {
		name := o.Number
		out = append(out, *apiresource.NewEntity(o.Id, constants.ObjectTypePurchaseOrder, &name, nonEmpty(o.SupplierName)))
	}
	return out, nil
}

func (m *searchSvcImpl) searchInvoices(ctx context.Context, query string, limit int32, scope searchScope) ([]apiresource.Entity, *apierror.APIError) {
	req := &corepb.ListInvoicesRequest{Query: &query, Limit: limit}
	if scope.customerID != "" {
		req.CustomerIds = []string{scope.customerID}
	}
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.invoices", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.ListInvoicesResponse, error) {
			return m.coreClient.ListInvoices(ctx, req, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if resp == nil {
		return nil, nil
	}
	out := make([]apiresource.Entity, 0, len(resp.Invoices))
	for _, inv := range resp.Invoices {
		name := inv.Number
		out = append(out, *apiresource.NewEntity(inv.Id, constants.ObjectTypeInvoice, &name, nonEmpty(inv.CustomerName)))
	}
	return out, nil
}

func (m *searchSvcImpl) searchCustomers(ctx context.Context, query string, limit int32, _ searchScope) ([]apiresource.Entity, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.customers", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.ListCustomersResponse, error) {
			return m.coreClient.ListCustomers(ctx, &corepb.ListCustomersRequest{Query: &query, Limit: limit}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	if resp == nil {
		return nil, nil
	}
	out := make([]apiresource.Entity, 0, len(resp.Customers))
	for _, c := range resp.Customers {
		name := c.Name
		out = append(out, *apiresource.NewEntity(c.Id, constants.ObjectTypeCustomer, &name, nonEmpty(c.Number)))
	}
	return out, nil
}

func (m *searchSvcImpl) searchItems(ctx context.Context, query string, limit int32, _ searchScope) ([]apiresource.Entity, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.items", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.ListItemsResponse, error) {
			return m.coreClient.ListItems(ctx, &corepb.ListItemsRequest{Query: &query, Limit: limit}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]apiresource.Entity, 0, len(resp.Items))
	for _, it := range resp.Items {
		name := it.Sku
		out = append(out, *apiresource.NewEntity(it.Id, constants.ObjectTypeItem, &name, ptrNonEmpty(it.Description)))
	}
	return out, nil
}

func (m *searchSvcImpl) searchProducts(ctx context.Context, query string, limit int32, _ searchScope) ([]apiresource.Entity, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.products", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.ListProductsFullResponse, error) {
			return m.coreClient.ListProductsFull(ctx, &corepb.ListProductsFullRequest{Query: &query, Limit: limit, Includes: []string{"item"}}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]apiresource.Entity, 0, len(resp.Products))
	for _, p := range resp.Products {
		// A product's display name is its item's SKU; skip products whose item did not resolve rather than emit a nameless reference.
		if p.Item == nil || p.Item.Sku == "" {
			continue
		}
		name := p.Item.Sku
		out = append(out, *apiresource.NewEntity(p.Id, constants.ObjectTypeProduct, &name, ptrNonEmpty(p.Item.Description)))
	}
	return out, nil
}

func (m *searchSvcImpl) searchShipments(ctx context.Context, query string, limit int32, scope searchScope) ([]apiresource.Entity, *apierror.APIError) {
	req := &corepb.ListShipmentsRequest{Query: &query, Limit: limit}
	if scope.customerID != "" {
		req.CustomerIds = []string{scope.customerID}
	}
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.shipments", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.ListShipmentsResponse, error) {
			return m.shippingClient.ListShipments(ctx, req, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]apiresource.Entity, 0, len(resp.Shipments))
	for _, s := range resp.Shipments {
		name := s.Number
		out = append(out, *apiresource.NewEntity(s.Id, constants.ObjectTypeShipment, &name, ptrNonEmpty(s.MasterTrackingNumber)))
	}
	return out, nil
}

func (m *searchSvcImpl) searchContacts(ctx context.Context, query string, _ int32, _ searchScope) ([]apiresource.Entity, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.contacts", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*notifpb.ListContactsResponse, error) {
			return m.chatClient.ListContacts(ctx, &notifpb.ListContactsRequest{Query: query}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]apiresource.Entity, 0, len(resp.Contacts))
	for _, c := range resp.Contacts {
		// A contact is addressable only when it maps to an account user; skip pure external contacts that have no linkable id.
		if c.AccountUserId == nil || *c.AccountUserId == "" {
			continue
		}
		name := c.Name
		out = append(out, *apiresource.NewEntity(*c.AccountUserId, constants.ObjectTypeMessagingContact, &name, nonEmpty(c.Type)))
	}
	return out, nil
}

func (m *searchSvcImpl) searchAgents(ctx context.Context, query string, limit int32, _ searchScope) ([]apiresource.Entity, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, searchSvcTracer, "service.search.agents", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*agentpb.ListAgentDefinitionsResponse, error) {
			return m.agentClient.ListAgentDefinitions(ctx, &agentpb.ListAgentDefinitionsRequest{Query: &query, Limit: limit}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make([]apiresource.Entity, 0, len(resp.Agents))
	for _, a := range resp.Agents {
		name := a.Name
		out = append(out, *apiresource.NewEntity(a.Id, constants.ObjectTypeAgentDefinition, &name, nonEmpty(a.Slug)))
	}
	return out, nil
}

// nonEmpty returns a pointer to s, or nil when s is empty, for optional Entity handle fields.
func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ptrNonEmpty normalizes an optional proto string to a handle pointer, collapsing both nil and empty to nil.
func ptrNonEmpty(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}
