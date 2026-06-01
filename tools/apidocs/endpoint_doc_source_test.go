package main

import (
	"fmt"
	"reflect"
	"testing"

	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	httpgroup "github.com/augno/api/services/api-gateway/pkg/group"
	agentpb "github.com/augno/api/shared/proto/agent"
	authpb "github.com/augno/api/shared/proto/auth"
	billingpb "github.com/augno/api/shared/proto/billing"
	pbgrpc "github.com/augno/api/shared/proto/core"
	platformpb "github.com/augno/api/shared/proto/platform"
)

// TestAllEndpointsHaveDocSource verifies that every endpoint registered in every
// group has EndpointType set (via apiendpoint.From). This guarantees that the
// OpenAPI generator can resolve documentation from the endpoint struct's Go doc
// comment. If this test fails, the endpoint was registered with a raw
// .Materialize() call instead of apiendpoint.From(&YourEndpoint{}).
func TestAllEndpointsHaveDocSource(t *testing.T) {
	groups := buildAllGroups()

	var missing []string
	for _, group := range groups {
		for _, ep := range group.Endpoints {
			val := reflect.ValueOf(ep)
			if val.Kind() == reflect.Pointer {
				val = val.Elem()
			}

			specField := val.FieldByName("APIEndpoint")
			if !specField.IsValid() {
				specField = val
			}

			title := specField.FieldByName("Title").String()
			method := specField.FieldByName("Method").String()
			route := specField.FieldByName("Route").String()

			epTypeVal := specField.FieldByName("EndpointType")
			if !epTypeVal.IsValid() || epTypeVal.IsNil() {
				missing = append(missing, fmt.Sprintf("[%s] %s %s (%s)", group.Title, method, route, title))
			}
		}
	}

	if len(missing) > 0 {
		t.Errorf("The following %d endpoint(s) are missing a doc source (EndpointType is nil).\n"+
			"Fix: use apiendpoint.From(&YourEndpoint{}) in the group instead of (&YourEndpoint{}).Materialize().\n\n", len(missing))
		for _, m := range missing {
			t.Errorf("  %s", m)
		}
	}
}

// TestAllEndpointsHaveService verifies that every endpoint registered in every
// group has been bound to a service via .WithService(). Without this call the
// endpoint handler will panic at runtime when it attempts to invoke the service
// method. If this test fails, add .WithService(inner, svc) to the endpoint
// registration chain in the group file.
func TestAllEndpointsHaveService(t *testing.T) {
	groups := buildAllGroups()

	var missing []string
	for _, group := range groups {
		for _, ep := range group.Endpoints {
			if !ep.IsServiceBound() {
				val := reflect.ValueOf(ep)
				if val.Kind() == reflect.Pointer {
					val = val.Elem()
				}
				specField := val.FieldByName("APIEndpoint")
				if !specField.IsValid() {
					specField = val
				}
				title := specField.FieldByName("Title").String()
				method := specField.FieldByName("Method").String()
				route := specField.FieldByName("Route").String()
				missing = append(missing, fmt.Sprintf("[%s] %s %s (%s)", group.Title, method, route, title))
			}
		}
	}

	if len(missing) > 0 {
		t.Errorf("The following %d endpoint(s) are missing a service binding.\n"+
			"Fix: add .WithService(inner, svc) to the endpoint registration in the group file.\n\n", len(missing))
		for _, m := range missing {
			t.Errorf("  %s", m)
		}
	}
}

func buildAllGroups() []apiendpoint.APIEndpointGroup {
	authClient := &grpcclient.AuthServiceClient{
		Client: struct{ authpb.AuthServiceClient }{},
	}
	coreClient := &grpcclient.CoreServiceClient{
		Client: struct{ pbgrpc.CoreServiceClient }{},
		Sales:  struct{ pbgrpc.CoreSalesServiceClient }{},
		Purchase: struct {
			pbgrpc.CorePurchaseServiceClient
		}{},
		Fulfillment: struct {
			pbgrpc.CoreFulfillmentServiceClient
		}{},
		Picking: struct {
			pbgrpc.CorePickingServiceClient
		}{},
		ProductionRun: struct {
			pbgrpc.CoreProductionRunServiceClient
		}{},
		ProductionStep: struct {
			pbgrpc.CoreProductionStepServiceClient
		}{},
		Receiving: struct {
			pbgrpc.CoreReceivingServiceClient
		}{},
		Shipping: struct {
			pbgrpc.CoreShippingServiceClient
		}{},
		ShippingCase: struct {
			pbgrpc.CoreShippingCaseServiceClient
		}{},
	}
	billingClient := &grpcclient.BillingServiceClient{
		Client: struct{ billingpb.BillingServiceClient }{},
	}
	platformClient := &grpcclient.PlatformServiceClient{
		Client: struct {
			platformpb.IdempotencyServiceClient
		}{},
		LoggingClient: struct {
			platformpb.LoggingServiceClient
		}{},
		AuditClient: struct {
			platformpb.AuditServiceClient
		}{},
	}
	agentClient := &grpcclient.AgentServiceClient{
		Client: struct{ agentpb.AgentServiceClient }{},
	}

	return []apiendpoint.APIEndpointGroup{
		*(&httpgroup.HealthEndpointGroup{}).Materialize(httpgroup.HealthEndpointGroupConfig{}).APIEndpointGroup,
		*(&httpgroup.AuthEndpointGroup{}).Materialize(&httpgroup.AuthEndpointGroupConfig{
			AuthClient: authClient,
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.APIKeysEndpointGroup{}).Materialize(&httpgroup.APIKeysEndpointGroupConfig{
			AuthClient: authClient,
		}).APIEndpointGroup,
		*(&httpgroup.TenancyEndpointGroup{}).Materialize(&httpgroup.TenancyEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.SandboxesEndpointGroup{}).Materialize(&httpgroup.SandboxesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.BillingEndpointGroup{}).Materialize(&httpgroup.BillingEndpointGroupConfig{
			BillingClient: billingClient,
			CoreClient:    coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.CheckoutSessionsEndpointGroup{}).Materialize(&httpgroup.CheckoutSessionsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.RegistrationSessionsEndpointGroup{}).Materialize(&httpgroup.RegistrationSessionsEndpointGroupConfig{
			AuthClient: authClient,
		}).APIEndpointGroup,
		*(&httpgroup.RegistrationFlowsEndpointGroup{}).Materialize(&httpgroup.RegistrationFlowsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.RequestLogsEndpointGroup{}).Materialize(&httpgroup.RequestLogsEndpointGroupConfig{
			PlatformClient: platformClient,
		}).APIEndpointGroup,
		*(&httpgroup.AuditEventsEndpointGroup{}).Materialize(&httpgroup.AuditEventsEndpointGroupConfig{
			PlatformClient: platformClient,
		}).APIEndpointGroup,
		*(&httpgroup.SysPropertiesEndpointGroup{}).Materialize(&httpgroup.SysPropertiesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.UnitsEndpointGroup{}).Materialize(&httpgroup.UnitsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.UnitGroupsEndpointGroup{}).Materialize(&httpgroup.UnitGroupsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AgentsEndpointGroup{}).Materialize(&httpgroup.AgentsEndpointGroupConfig{
			AgentClient: agentClient,
			CoreClient:  coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AgentAlertsEndpointGroup{}).Materialize(&httpgroup.AgentAlertsEndpointGroupConfig{
			AgentClient: agentClient,
		}).APIEndpointGroup,
		*(&httpgroup.AgentRunsEndpointGroup{}).Materialize(&httpgroup.AgentRunsEndpointGroupConfig{
			AgentClient: agentClient,
		}).APIEndpointGroup,
		*(&httpgroup.AgentToolsEndpointGroup{}).Materialize(&httpgroup.AgentToolsEndpointGroupConfig{
			AgentClient: agentClient,
		}).APIEndpointGroup,
		*(&httpgroup.AgentMemoriesEndpointGroup{}).Materialize(&httpgroup.AgentMemoriesEndpointGroupConfig{
			AgentClient: agentClient,
		}).APIEndpointGroup,
		*(&httpgroup.WebhooksEndpointGroup{}).Materialize(&httpgroup.WebhooksEndpointGroupConfig{
			BillingClient: billingClient,
		}).APIEndpointGroup,
		*(&httpgroup.AccountGroupsEndpointGroup{}).Materialize(&httpgroup.AccountGroupsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AccountGroupProductLineAccessEndpointGroup{}).Materialize(&httpgroup.AccountGroupProductLineAccessEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AccountPricesEndpointGroup{}).Materialize(&httpgroup.AccountPricesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.PaymentTermsEndpointGroup{}).Materialize(&httpgroup.PaymentTermsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ShippingTermsEndpointGroup{}).Materialize(&httpgroup.ShippingTermsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AddressesEndpointGroup{}).Materialize(&httpgroup.AddressesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AddressValidationEndpointGroup{}).Materialize(&httpgroup.AddressValidationEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AccountStatusesEndpointGroup{}).Materialize(&httpgroup.AccountStatusesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AccountUsersEndpointGroup{}).Materialize(&httpgroup.AccountUsersEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.SalesTargetsEndpointGroup{}).Materialize(&httpgroup.SalesTargetsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.UsersEndpointGroup{}).Materialize(&httpgroup.UsersEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.PropertiesEndpointGroup{}).Materialize(&httpgroup.PropertiesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AccountsEndpointGroup{}).Materialize(&httpgroup.AccountsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AccountIntegrationsEndpointGroup{}).Materialize(&httpgroup.AccountIntegrationsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.PrioritiesEndpointGroup{}).Materialize(&httpgroup.PrioritiesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.CarriersEndpointGroup{}).Materialize(&httpgroup.CarriersEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ServiceLevelsEndpointGroup{}).Materialize(&httpgroup.ServiceLevelsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ChildAccountsEndpointGroup{}).Materialize(&httpgroup.ChildAccountsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ItemsEndpointGroup{}).Materialize(&httpgroup.ItemsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.InventoriesEndpointGroup{}).Materialize(&httpgroup.InventoriesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ItemCategoriesEndpointGroup{}).Materialize(&httpgroup.ItemCategoriesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.MaterialsEndpointGroup{}).Materialize(&httpgroup.MaterialsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.SupplierMaterialsEndpointGroup{}).Materialize(&httpgroup.SupplierMaterialsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.PartsEndpointGroup{}).Materialize(&httpgroup.PartsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.BatchesEndpointGroup{}).Materialize(&httpgroup.BatchesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.DepartmentsEndpointGroup{}).Materialize(&httpgroup.DepartmentsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ConsumptionsEndpointGroup{}).Materialize(&httpgroup.ConsumptionsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.DeliveriesEndpointGroup{}).Materialize(&httpgroup.DeliveriesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.EmailLogsEndpointGroup{}).Materialize(&httpgroup.EmailLogsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.SuppliersEndpointGroup{}).Materialize(&httpgroup.SuppliersEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.CustomersEndpointGroup{}).Materialize(&httpgroup.CustomersEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.CustomerProductLineAccessEndpointGroup{}).Materialize(&httpgroup.CustomerProductLineAccessEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.InvoicesEndpointGroup{}).Materialize(&httpgroup.InvoicesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.InventoryChangeLogsEndpointGroup{}).Materialize(&httpgroup.InventoryChangeLogsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ReceivablesEndpointGroup{}).Materialize(&httpgroup.ReceivablesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.MachinesEndpointGroup{}).Materialize(&httpgroup.MachinesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.OrderDiscountsEndpointGroup{}).Materialize(&httpgroup.OrderDiscountsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ProductLinesEndpointGroup{}).Materialize(&httpgroup.ProductLinesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ProductsEndpointGroup{}).Materialize(&httpgroup.ProductsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ProductTypesEndpointGroup{}).Materialize(&httpgroup.ProductTypesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.SalesOrderStatusesEndpointGroup{}).Materialize(&httpgroup.SalesOrderStatusesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ReceivingOrdersEndpointGroup{}).Materialize(&httpgroup.ReceivingOrdersEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ProductionStepsEndpointGroup{}).Materialize(&httpgroup.ProductionStepsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ProductionFlowsEndpointGroup{}).Materialize(&httpgroup.ProductionFlowsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ProductionRunsEndpointGroup{}).Materialize(&httpgroup.ProductionRunsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.VolumeDiscountsEndpointGroup{}).Materialize(&httpgroup.VolumeDiscountsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.SalesOrdersEndpointGroup{}).Materialize(&httpgroup.SalesOrdersEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.PurchaseOrdersEndpointGroup{}).Materialize(&httpgroup.PurchaseOrdersEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.PicksEndpointGroup{}).Materialize(&httpgroup.PicksEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.TransactionsEndpointGroup{}).Materialize(&httpgroup.TransactionsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.TransactionAllocationsEndpointGroup{}).Materialize(&httpgroup.TransactionAllocationsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.SettlementsEndpointGroup{}).Materialize(&httpgroup.SettlementsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.TerritoriesEndpointGroup{}).Materialize(&httpgroup.TerritoriesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.LocationsEndpointGroup{}).Materialize(&httpgroup.LocationsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ShippingCasesEndpointGroup{}).Materialize(&httpgroup.ShippingCasesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ShipmentsEndpointGroup{}).Materialize(&httpgroup.ShipmentsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.ScanningStationsEndpointGroup{}).Materialize(&httpgroup.ScanningStationsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.RolesEndpointGroup{}).Materialize(&httpgroup.RolesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.PermissionGroupsEndpointGroup{}).Materialize(&httpgroup.PermissionGroupsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.QuantitiesEndpointGroup{}).Materialize(&httpgroup.QuantitiesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.RatesEndpointGroup{}).Materialize(&httpgroup.RatesEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.EDIEndpointGroup{}).Materialize(&httpgroup.EDIEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.EDIDCLocationsEndpointGroup{}).Materialize(&httpgroup.EDIDCLocationsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.EDIRunsEndpointGroup{}).Materialize(&httpgroup.EDIRunsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.CatalogEndpointGroup{}).Materialize(&httpgroup.CatalogEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.AnalyticsEndpointGroup{}).Materialize(&httpgroup.AnalyticsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
		*(&httpgroup.UtilsEndpointGroup{}).Materialize(&httpgroup.UtilsEndpointGroupConfig{
			CoreClient: coreClient,
		}).APIEndpointGroup,
	}
}
