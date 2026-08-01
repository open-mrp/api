// Package resourceloaders holds the gRPC-backed loaders used by the api-gateway resourcekit include resolver. Each loader is a plain function that takes a slice of IDs and returns a map[id]*apiresource (typed as any to fit resourcekit's signature). Loaders build clean apiresource values and stash any FK metadata the SubField closures will need into the request-scoped resourcekit.LoadMeta side-table.
//
// The core client is set once at process startup via SetCoreClient — the loaders cannot be invoked before that.
package resourceloaders

import (
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	agentpb "github.com/augno/api/shared/proto/agent"
	authpb "github.com/augno/api/shared/proto/auth"
	pb "github.com/augno/api/shared/proto/core"
	notifpb "github.com/augno/api/shared/proto/notification"
	platformpb "github.com/augno/api/shared/proto/platform"
)

// coreClient is the primary CoreService gRPC client used by every loader whose RPC lives on CoreService (carrier, account, priority, etc.). Set by SetCoreClient during api-gateway startup before any HTTP traffic is served.
var coreClient pb.CoreServiceClient

// coreSalesClient is the CoreSalesService client (different proto service) used by loaders whose RPCs live in core_sales.proto (sales-order-status, sales-order, order-discount, etc.). Set together with coreClient via SetCoreClient(s).
var coreSalesClient pb.CoreSalesServiceClient

// agentClient is the AgentService client used by loaders whose RPCs live in agent.proto (agent-memory, agent-run, agent-alert, agent-tool, etc.). Set at startup alongside coreClient.
var agentClient agentpb.AgentServiceClient

// SetCoreClient is called once from the api-gateway wiring code with the shared CoreServiceClient wrapper. Calling it more than once overwrites the previous value — useful for tests that swap in mocks.
func SetCoreClient(c pb.CoreServiceClient) {
	coreClient = c
}

// SetCoreSalesClient is called once at startup with the CoreSalesService client.
func SetCoreSalesClient(c pb.CoreSalesServiceClient) {
	coreSalesClient = c
}

// corePurchaseClient is the CorePurchaseService client used by loaders whose RPCs live in core_purchase.proto (purchase-order include resolution).
var corePurchaseClient pb.CorePurchaseServiceClient

// SetCorePurchaseClient is called once at startup with the CorePurchaseService client.
func SetCorePurchaseClient(c pb.CorePurchaseServiceClient) {
	corePurchaseClient = c
}

// SetAgentClient is called once at startup with the AgentService client.
func SetAgentClient(c agentpb.AgentServiceClient) {
	agentClient = c
}

// fulfillmentClient is the CoreFulfillmentService client used by loaders whose RPCs live in core_fulfillment.proto (machine, etc.). Set at startup.
var fulfillmentClient pb.CoreFulfillmentServiceClient

// SetFulfillmentClient is called once at startup with the CoreFulfillmentService client.
func SetFulfillmentClient(c pb.CoreFulfillmentServiceClient) {
	fulfillmentClient = c
}

// machineDowntimeClient is the CoreMachineDowntimeService client used by LoadMachineDowntimeEvents. Set at startup.
var machineDowntimeClient pb.CoreMachineDowntimeServiceClient

// SetMachineDowntimeClient is called once at startup with the CoreMachineDowntimeService client.
func SetMachineDowntimeClient(c pb.CoreMachineDowntimeServiceClient) {
	machineDowntimeClient = c
}

// demandOverrideClient is the CoreDemandOverrideService client used by LoadDemandOverrides. Set at startup.
var demandOverrideClient pb.CoreDemandOverrideServiceClient

// SetDemandOverrideClient is called once at startup with the CoreDemandOverrideService client.
func SetDemandOverrideClient(c pb.CoreDemandOverrideServiceClient) {
	demandOverrideClient = c
}

// corePickingClient is the CorePickingService client used by LoadPicks (pick include resolution). Set at startup.
var corePickingClient pb.CorePickingServiceClient

// SetCorePickingClient is called once at startup with the CorePickingService client.
func SetCorePickingClient(c pb.CorePickingServiceClient) {
	corePickingClient = c
}

// portalDomainClient is the CorePortalDomainService client used by LoadPortalDomains. Set at startup.
var portalDomainClient pb.CorePortalDomainServiceClient

// SetPortalDomainClient is called once at startup with the CorePortalDomainService client.
func SetPortalDomainClient(c pb.CorePortalDomainServiceClient) {
	portalDomainClient = c
}

// coreShippingClient is the CoreShippingService client used by LoadShipments (shipment include resolution). Set at startup.
var coreShippingClient pb.CoreShippingServiceClient

// SetCoreShippingClient is called once at startup with the CoreShippingService client.
func SetCoreShippingClient(c pb.CoreShippingServiceClient) {
	coreShippingClient = c
}

// coreReceivingClient is the CoreReceivingService client used by LoadReceivingOrders (receiving order include resolution). Set at startup.
var coreReceivingClient pb.CoreReceivingServiceClient

// SetCoreReceivingClient is called once at startup with the CoreReceivingService client.
func SetCoreReceivingClient(c pb.CoreReceivingServiceClient) {
	coreReceivingClient = c
}

// coreProductionRunClient is the CoreProductionRunService client used by LoadProductionRuns (the sales order's related.production_run reference). Set at startup.
var coreProductionRunClient pb.CoreProductionRunServiceClient

// SetCoreProductionRunClient is called once at startup with the CoreProductionRunService client.
func SetCoreProductionRunClient(c pb.CoreProductionRunServiceClient) {
	coreProductionRunClient = c
}

// authClient is the AuthService client used by loaders whose RPCs live in auth.proto (api-key, etc.). Set at startup alongside coreClient.
var authClient authpb.AuthServiceClient

// SetAuthClient is called once at startup with the AuthService client.
func SetAuthClient(c authpb.AuthServiceClient) {
	authClient = c
}

// loggingClient is the LoggingService client used by loaders whose RPCs live in platform.proto (request-log, etc.). Set at startup.
var loggingClient platformpb.LoggingServiceClient

// SetLoggingClient is called once at startup with the LoggingService client.
func SetLoggingClient(c platformpb.LoggingServiceClient) {
	loggingClient = c
}

// auditClient is the AuditService client used by LoadCreatedBySalesOrders to resolve a resource's creator from its create audit event. Set at startup.
var auditClient platformpb.AuditServiceClient

// SetAuditClient is called once at startup with the AuditService client.
func SetAuditClient(c platformpb.AuditServiceClient) {
	auditClient = c
}

// chatClient is the notification-service ChatService client used by LoadConversations / LoadMessages to resolve a message's expandable conversation and reply_to references. Set at startup.
var chatClient notifpb.ChatServiceClient

// SetChatClient is called once at startup with the ChatService client.
func SetChatClient(c notifpb.ChatServiceClient) {
	chatClient = c
}

// emailBridgeClient is the notification-service EmailBridgeService client used by LoadEmailDomains to resolve an email inbox's expandable email_domain reference. Set at startup.
var emailBridgeClient notifpb.EmailBridgeServiceClient

// SetEmailBridgeClient is called once at startup with the EmailBridgeService client.
func SetEmailBridgeClient(c notifpb.EmailBridgeServiceClient) {
	emailBridgeClient = c
}

// orEmptyStrSlice normalizes a nil slice to a non-nil empty slice so required array fields serialize as `[]` rather than `null`.
func orEmptyStrSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ownerShellFromAccountID builds an Owner shell without a stub Account. nil/empty → system-owned; non-empty → account-owned with Account left nil (populated only when owner.account is explicitly included).
func ownerShellFromAccountID(accountID *string) *apiresource.Owner {
	if accountID == nil || *accountID == "" {
		return &apiresource.Owner{
			Object: constants.ObjectTypeOwner,
			Type:   constants.OwnerTypeSystem,
		}
	}
	return &apiresource.Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeAccount,
	}
}
