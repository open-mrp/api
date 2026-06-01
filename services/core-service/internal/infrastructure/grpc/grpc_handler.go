package grpc

import (
	"context"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
	pb "github.com/augno/api/shared/proto/core"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type gRPCHandler struct {
	pb.UnimplementedCoreServiceServer
	pb.UnimplementedCoreAccountServiceServer

	accountSvc                       domain.AccountSvc
	sandboxSvc                       domain.SandboxSvc
	unitSvc                          domain.UnitSvc
	paymentTermSvc                   domain.PaymentTermSvc
	shippingTermSvc                  domain.ShippingTermSvc
	accountStatusSvc                 domain.AccountStatusSvc
	accountGroupSvc                  domain.AccountGroupSvc
	accountGroupProductLineAccessSvc domain.AccountGroupProductLineAccessSvc
	customerProductLineAccessSvc     domain.CustomerProductLineAccessSvc
	productSvc                       domain.ProductSvc
	addressSvc                       domain.AddressSvc
	addressValidationSvc             domain.AddressValidationSvc
	accountUserSvc                   domain.AccountUserSvc
	accountPriceSvc                  domain.AccountPriceSvc
	salesTargetSvc                   domain.SalesTargetSvc
	accountIntegrationSvc            domain.AccountIntegrationSvc
	adjustmentTypeSvc                domain.AdjustmentTypeSvc
	propertySvc                      domain.PropertySvc
	attributeSvc                     domain.AttributeSvc
	carrierSvc                       domain.CarrierSvc
	serviceLevelSvc                  domain.ServiceLevelSvc
	itemSvc                          domain.ItemSvc
	childAccountSvc                  domain.ChildAccountSvc
	batchSvc                         domain.BatchSvc
	itemCategorySvc                  domain.ItemCategorySvc
	consumptionSvc                   domain.ConsumptionSvc
	customerSvc                      domain.CustomerSvc
	machineSvc                       domain.MachineSvc
	departmentSvc                    domain.DepartmentSvc
	deliverySvc                      domain.DeliverySvc
	emailLogSvc                      domain.EmailLogSvc
	inventoryChangeLogSvc            domain.InventoryChangeLogSvc
	invoiceSvc                       domain.InvoiceSvc
	salesOrderStatusSvc              domain.SalesOrderStatusSvc
	orderDiscountSvc                 domain.OrderDiscountSvc
	materialSvc                      domain.MaterialSvc
	supplierMaterialSvc              domain.SupplierMaterialSvc
	partSvc                          domain.PartSvc
	permissionGroupSvc               domain.PermissionGroupSvc
	prioritySvc                      domain.PrioritySvc
	productLineSvc                   domain.ProductLineSvc
	productTypeSvc                   domain.ProductTypeSvc
	productionFlowSvc                domain.ProductionFlowSvc
	measureSvc                       domain.MeasureSvc
	receivableSvc                    domain.ReceivableSvc
	registrationFlowSvc              domain.RegistrationFlowSvc
	scanningStationSvc               domain.ScanningStationSvc
	settlementSvc                    domain.SettlementSvc
	locationSvc                      domain.LocationSvc
	supplierSvc                      domain.SupplierSvc
	sysPropertySvc                   domain.SysPropertySvc
	tenancySvc                       domain.TenancySvc
	userSvc                          domain.UserSvc
	territorySvc                     domain.TerritorySvc
	transactionAllocationSvc         domain.TransactionAllocationSvc
	transactionSvc                   domain.TransactionSvc
	unitGroupSvc                     domain.UnitGroupSvc
	utilsSvc                         domain.UtilsSvc
	analyticsSvc                     domain.AnalyticsSvc
	catalogSvc                       domain.CatalogSvc
	ediSvc                           domain.EDISvc
	roleSvc                          domain.RoleSvc
}

// handler is the shared gRPCHandler instance for the core service.
// All Register* functions populate fields on this shared handler.
var handler = &gRPCHandler{}

func RegisterAddressService(server *grpc.Server, addressSvc domain.AddressSvc, addressValidationSvc domain.AddressValidationSvc) {
	handler.addressSvc = addressSvc
	handler.addressValidationSvc = addressValidationSvc
}

func RegisterCarrierService(server *grpc.Server, carrierSvc domain.CarrierSvc, serviceLevelSvc domain.ServiceLevelSvc) {
	handler.carrierSvc = carrierSvc
	handler.serviceLevelSvc = serviceLevelSvc
}

func RegisterShippingCaseService(server *grpc.Server, shippingCaseSvc domain.ShippingCaseSvc) {
	shippingCaseHandler := &shippingCaseGRPCHandler{
		shippingCaseSvc: shippingCaseSvc,
	}
	pb.RegisterCoreShippingCaseServiceServer(server, shippingCaseHandler)
}

func RegisterMiscService(server *grpc.Server, accountIntegrationSvc domain.AccountIntegrationSvc, adjustmentTypeSvc domain.AdjustmentTypeSvc, emailLogSvc domain.EmailLogSvc, inventoryChangeLogSvc domain.InventoryChangeLogSvc, prioritySvc domain.PrioritySvc) {
	handler.accountIntegrationSvc = accountIntegrationSvc
	handler.adjustmentTypeSvc = adjustmentTypeSvc
	handler.emailLogSvc = emailLogSvc
	handler.inventoryChangeLogSvc = inventoryChangeLogSvc
	handler.prioritySvc = prioritySvc
}

func RegisterAccountUserService(server *grpc.Server, accountUserSvc domain.AccountUserSvc) {
	handler.accountUserSvc = accountUserSvc
}

func RegisterCustomerService(server *grpc.Server, customerSvc domain.CustomerSvc, childAccountSvc domain.ChildAccountSvc, customerProductLineAccessSvc domain.CustomerProductLineAccessSvc, productSvc domain.ProductSvc) {
	handler.customerSvc = customerSvc
	handler.childAccountSvc = childAccountSvc
	handler.customerProductLineAccessSvc = customerProductLineAccessSvc
	handler.productSvc = productSvc
}

func RegisterAnalyticsService(server *grpc.Server, analyticsSvc domain.AnalyticsSvc) {
	handler.analyticsSvc = analyticsSvc
}

func RegisterCatalogService(server *grpc.Server, catalogSvc domain.CatalogSvc) {
	handler.catalogSvc = catalogSvc
}

func RegisterEDIService(server *grpc.Server, ediSvc domain.EDISvc) {
	handler.ediSvc = ediSvc
}

func RegisterRoleService(server *grpc.Server, roleSvc domain.RoleSvc) {
	handler.roleSvc = roleSvc
}

func RegisterGroupService(server *grpc.Server, accountGroupSvc domain.AccountGroupSvc, accountGroupProductLineAccessSvc domain.AccountGroupProductLineAccessSvc) {
	handler.accountGroupSvc = accountGroupSvc
	handler.accountGroupProductLineAccessSvc = accountGroupProductLineAccessSvc
}

func RegisterSalesService(server *grpc.Server, accountPriceSvc domain.AccountPriceSvc, salesTargetSvc domain.SalesTargetSvc, productSvc domain.ProductSvc, invoiceSvc domain.InvoiceSvc, salesOrderStatusSvc domain.SalesOrderStatusSvc, orderDiscountSvc domain.OrderDiscountSvc, volumeDiscountSvc domain.VolumeDiscountSvc, salesOrderSvc domain.SalesOrderSvc, salesOrderLineSvc domain.SalesOrderLineSvc, receivableSvc domain.ReceivableSvc, settlementSvc domain.SettlementSvc, transactionAllocationSvc domain.TransactionAllocationSvc, transactionSvc domain.TransactionSvc) {
	handler.accountPriceSvc = accountPriceSvc
	handler.salesTargetSvc = salesTargetSvc
	handler.productSvc = productSvc
	handler.invoiceSvc = invoiceSvc
	handler.salesOrderStatusSvc = salesOrderStatusSvc
	handler.orderDiscountSvc = orderDiscountSvc
	handler.receivableSvc = receivableSvc
	handler.settlementSvc = settlementSvc
	handler.transactionAllocationSvc = transactionAllocationSvc
	handler.transactionSvc = transactionSvc

	salesHandler := &salesGRPCHandler{
		salesOrderStatusSvc: salesOrderStatusSvc,
		orderDiscountSvc:    orderDiscountSvc,
		volumeDiscountSvc:   volumeDiscountSvc,
		salesOrderSvc:       salesOrderSvc,
		salesOrderLineSvc:   salesOrderLineSvc,
	}
	pb.RegisterCoreSalesServiceServer(server, salesHandler)
}

func RegisterPurchaseService(server *grpc.Server, purchaseOrderSvc domain.PurchaseOrderSvc, purchaseOrderLineSvc domain.PurchaseOrderLineSvc) {
	purchaseHandler := &purchaseGRPCHandler{
		purchaseOrderSvc:     purchaseOrderSvc,
		purchaseOrderLineSvc: purchaseOrderLineSvc,
	}
	pb.RegisterCorePurchaseServiceServer(server, purchaseHandler)
}

func RegisterFulfillmentService(server *grpc.Server, batchSvc domain.BatchSvc, consumptionSvc domain.ConsumptionSvc, deliverySvc domain.DeliverySvc, departmentSvc domain.DepartmentSvc, machineSvc domain.MachineSvc, productionFlowSvc domain.ProductionFlowSvc) {
	handler.batchSvc = batchSvc
	handler.consumptionSvc = consumptionSvc
	handler.deliverySvc = deliverySvc
	handler.departmentSvc = departmentSvc
	handler.machineSvc = machineSvc
	handler.productionFlowSvc = productionFlowSvc

	fulfillmentHandler := &fulfillmentGRPCHandler{
		machineSvc: machineSvc,
	}
	pb.RegisterCoreFulfillmentServiceServer(server, fulfillmentHandler)
}

func RegisterItemService(server *grpc.Server, unitSvc domain.UnitSvc, unitGroupSvc domain.UnitGroupSvc, itemSvc domain.ItemSvc, itemCategorySvc domain.ItemCategorySvc, propertySvc domain.PropertySvc, attributeSvc domain.AttributeSvc, paymentTermSvc domain.PaymentTermSvc, shippingTermSvc domain.ShippingTermSvc, partSvc domain.PartSvc, productLineSvc domain.ProductLineSvc, productTypeSvc domain.ProductTypeSvc) {
	handler.unitSvc = unitSvc
	handler.unitGroupSvc = unitGroupSvc
	handler.itemSvc = itemSvc
	handler.partSvc = partSvc
	handler.itemCategorySvc = itemCategorySvc
	handler.propertySvc = propertySvc
	handler.attributeSvc = attributeSvc
	handler.paymentTermSvc = paymentTermSvc
	handler.shippingTermSvc = shippingTermSvc
	handler.productLineSvc = productLineSvc
	handler.productTypeSvc = productTypeSvc
}

func RegisterMaterialService(server *grpc.Server, materialSvc domain.MaterialSvc, supplierMaterialSvc domain.SupplierMaterialSvc) {
	handler.materialSvc = materialSvc
	handler.supplierMaterialSvc = supplierMaterialSvc
}

func RegisterPermissionGroupService(server *grpc.Server, permissionGroupSvc domain.PermissionGroupSvc) {
	handler.permissionGroupSvc = permissionGroupSvc
}

func RegisterPickingService(server *grpc.Server, pickSvc domain.PickSvc, pickLineSvc domain.PickLineSvc) {
	pickingHandler := &pickingGRPCHandler{
		pickSvc:     pickSvc,
		pickLineSvc: pickLineSvc,
	}
	pb.RegisterCorePickingServiceServer(server, pickingHandler)
}

func RegisterReceivingService(server *grpc.Server, receivingOrderSvc domain.ReceivingOrderSvc, receivingOrderLineSvc domain.ReceivingOrderLineSvc) {
	receivingHandler := &receivingGRPCHandler{
		receivingOrderSvc:     receivingOrderSvc,
		receivingOrderLineSvc: receivingOrderLineSvc,
	}
	pb.RegisterCoreReceivingServiceServer(server, receivingHandler)
}

func RegisterProductionRunService(server *grpc.Server, productionRunSvc domain.ProductionRunSvc) {
	productionRunHandler := &productionRunGRPCHandler{
		productionRunSvc: productionRunSvc,
	}
	pb.RegisterCoreProductionRunServiceServer(server, productionRunHandler)
}

func RegisterUserService(server *grpc.Server, userSvc domain.UserSvc) {
	handler.userSvc = userSvc
}

func RegisterTenancyService(server *grpc.Server, tenancySvc domain.TenancySvc) {
	handler.tenancySvc = tenancySvc
}

func RegisterUtilsService(server *grpc.Server, utilsSvc domain.UtilsSvc) {
	handler.utilsSvc = utilsSvc
}

func RegisterLocationService(server *grpc.Server, locationSvc domain.LocationSvc) {
	handler.locationSvc = locationSvc
}

func RegisterScanningStationService(server *grpc.Server, scanningStationSvc domain.ScanningStationSvc) {
	handler.scanningStationSvc = scanningStationSvc
}

func RegisterSupplierService(server *grpc.Server, supplierSvc domain.SupplierSvc) {
	handler.supplierSvc = supplierSvc
}

func RegisterSysPropertyService(server *grpc.Server, sysPropertySvc domain.SysPropertySvc) {
	handler.sysPropertySvc = sysPropertySvc
}

func RegisterRegistrationFlowService(server *grpc.Server, registrationFlowSvc domain.RegistrationFlowSvc) {
	handler.registrationFlowSvc = registrationFlowSvc
}

func RegisterShippingService(server *grpc.Server, shipmentSvc domain.ShipmentSvc, shipmentLineSvc domain.ShipmentLineSvc) {
	shippingHandler := &shippingGRPCHandler{
		shipmentSvc:     shipmentSvc,
		shipmentLineSvc: shipmentLineSvc,
	}
	pb.RegisterCoreShippingServiceServer(server, shippingHandler)
}

func RegisterAccountService(server *grpc.Server, accountSvc domain.AccountSvc, sandboxSvc domain.SandboxSvc, accountStatusSvc domain.AccountStatusSvc) {
	handler.accountSvc = accountSvc
	handler.sandboxSvc = sandboxSvc
	handler.accountStatusSvc = accountStatusSvc

	// RegisterAccountService is called last — register the core service handler.
	pb.RegisterCoreServiceServer(server, handler)
	pb.RegisterCoreAccountServiceServer(server, handler)
}

func (h *gRPCHandler) GetAccountContext(ctx context.Context, req *pb.GetAccountContextRequest) (*pb.GetAccountContextResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	accountContext, apiErr := h.accountSvc.GetAccountContext(ctx, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	var accountMode pb.AccountMode
	switch accountContext.AccountMode {
	case constants.AccountModeProduction:
		accountMode = pb.AccountMode_ACCOUNT_MODE_PRODUCTION
	case constants.AccountModeSandbox:
		accountMode = pb.AccountMode_ACCOUNT_MODE_SANDBOX
	default:
		accountMode = pb.AccountMode_ACCOUNT_MODE_UNSPECIFIED
	}

	return &pb.GetAccountContextResponse{
		IsSandbox:                    accountContext.IsSandbox,
		OwnerAccountId:               accountContext.OwnerAccountID,
		AccountMode:                  accountMode,
		SubscriptionStatus:           accountContext.SubscriptionStatus,
		PlanCode:                     accountContext.PlanCode,
		AgentMonthlySpendingCapCents: accountContext.AgentMonthlySpendingCapCents,
	}, nil
}

func (h *gRPCHandler) GetUserAccountAccess(ctx context.Context, req *pb.GetUserAccountAccessRequest) (*pb.GetUserAccountAccessResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	access, hasAccess, apiErr := h.accountSvc.GetUserAccountAccess(ctx, req.UserId, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	if !hasAccess {
		return &pb.GetUserAccountAccessResponse{
			HasAccess: false,
		}, nil
	}

	var lastUsedAt *timestamppb.Timestamp
	if access.LastUsedAt != nil {
		lastUsedAt = timestamppb.New(*access.LastUsedAt)
	}

	return &pb.GetUserAccountAccessResponse{
		HasAccess: true,
		Access: &pb.AccountUserAccess{
			AccountUserId: access.AccountUserID,
			AccountId:     access.AccountID,
			RoleId:        access.RoleID,
			RoleTypeCode:  access.RoleType,
			Permissions:   access.Permissions,
			LastUsedAt:    lastUsedAt,
		},
	}, nil
}

func (h *gRPCHandler) GetRolePermissions(ctx context.Context, req *pb.GetRolePermissionsRequest) (*pb.GetRolePermissionsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	permissions, apiErr := h.accountSvc.GetRolePermissions(ctx, req.RoleId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRolePermissionsResponse{
		Permissions: permissions,
	}, nil
}

func (h *gRPCHandler) GetRoleInfo(ctx context.Context, req *pb.GetRoleInfoRequest) (*pb.GetRoleInfoResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	role, apiErr := h.accountSvc.GetRoleInfo(ctx, req.RoleId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRoleInfoResponse{
		RoleId:       role.ID,
		Name:         role.Name,
		RoleTypeCode: role.RoleType,
	}, nil
}

func (h *gRPCHandler) GetAccountRelation(ctx context.Context, req *pb.GetAccountRelationRequest) (*pb.GetAccountRelationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	var relation *domain.AccountRelation
	var apiErr *apierror.APIError

	switch lookup := req.Lookup.(type) {
	case *pb.GetAccountRelationRequest_UserId:
		actorAccountID := ""
		if req.ActorAccountId != nil {
			actorAccountID = *req.ActorAccountId
		}
		relation, apiErr = h.accountSvc.GetAccountRelationByUserID(ctx, req.OwnerAccountId, actorAccountID, lookup.UserId)
	case *pb.GetAccountRelationRequest_ApiKeyId:
		relation, apiErr = h.accountSvc.GetAccountRelationByAPIKeyID(ctx, req.OwnerAccountId, lookup.ApiKeyId)
	default:
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewParameterMissingError("Either user_id or api_key_id must be provided", "lookup"))
	}

	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	if relation == nil {
		return &pb.GetAccountRelationResponse{
			HasRelation: false,
			Relation:    nil,
		}, nil
	}

	return &pb.GetAccountRelationResponse{
		HasRelation: true,
		Relation: &pb.AccountRelation{
			Id:                    relation.ID,
			CounterpartyAccountId: relation.CounterpartyAccountID,
			RoleCode:              relation.RoleCode,
			IsOwnerSide:           relation.IsOwnerSide,
		},
	}, nil
}

func (h *gRPCHandler) MarkAccountUserUsed(ctx context.Context, req *pb.MarkAccountUserUsedRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountSvc.MarkAccountUserUsed(ctx, req.AccountUserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ListUserAccountAffiliations(ctx context.Context, req *pb.ListUserAccountAffiliationsRequest) (*pb.ListUserAccountAffiliationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	affiliations, lastUsedAccountID, apiErr := h.accountSvc.ListUserAccountAffiliations(ctx, req.UserId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbAffiliations := make([]*pb.AccountAffiliation, len(affiliations))
	for i, aff := range affiliations {
		var lastUsedAt *timestamppb.Timestamp
		if aff.LastUsedAt != nil {
			lastUsedAt = timestamppb.New(*aff.LastUsedAt)
		}

		pbAffiliations[i] = &pb.AccountAffiliation{
			AccountId:    aff.AccountID,
			AccountName:  aff.AccountName,
			RoleId:       aff.RoleID,
			RoleName:     aff.RoleName,
			RoleTypeCode: aff.RoleType,
			LastUsedAt:   lastUsedAt,
		}
	}

	return &pb.ListUserAccountAffiliationsResponse{
		Affiliations:      pbAffiliations,
		LastUsedAccountId: lastUsedAccountID,
	}, nil
}

func (h *gRPCHandler) GetSandboxAccountByOwner(ctx context.Context, req *pb.GetSandboxAccountByOwnerRequest) (*pb.GetSandboxAccountByOwnerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	sandboxAccountID, apiErr := h.sandboxSvc.GetSandboxAccountByOwner(ctx, req.OwnerAccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSandboxAccountByOwnerResponse{
		SandboxAccountId: sandboxAccountID,
	}, nil
}

func (h *gRPCHandler) ListSandboxAccounts(ctx context.Context, req *pb.ListSandboxAccountsRequest) (*pb.ListSandboxAccountsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.sandboxSvc.ListSandboxAccounts(ctx, req.Cursor, req.Limit, req.Query, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbSandboxes := make([]*pb.SandboxInfo, len(result.Sandboxes))
	for i, s := range result.Sandboxes {
		pbSandboxes[i] = sandboxToProto(s)
	}

	return &pb.ListSandboxAccountsResponse{
		Sandboxes: pbSandboxes,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetAdminRole(ctx context.Context, req *emptypb.Empty) (*pb.GetAdminRoleResponse, error) {
	roleID, apiErr := h.accountSvc.GetAdminRole(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAdminRoleResponse{
		RoleId: roleID,
	}, nil
}

func (h *gRPCHandler) GetSandbox(ctx context.Context, req *pb.GetSandboxRequest) (*pb.GetSandboxResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	sandbox, apiErr := h.sandboxSvc.GetSandbox(ctx, req.Id, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSandboxResponse{
		Sandbox: sandboxToProto(sandbox),
	}, nil
}

func (h *gRPCHandler) BatchGetSandboxesByIDs(ctx context.Context, req *pb.BatchGetSandboxesByIDsRequest) (*pb.BatchGetSandboxesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	sandboxes, apiErr := h.sandboxSvc.BatchGetSandboxesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbSandboxes := make([]*pb.SandboxInfo, len(sandboxes))
	for i, s := range sandboxes {
		pbSandboxes[i] = sandboxToProtoFlat(s)
	}
	return &pb.BatchGetSandboxesByIDsResponse{Sandboxes: pbSandboxes}, nil
}

// sandboxToProtoFlat returns the minimal SandboxInfo for V2 batch reads: it
// always populates owner_account_id (the FK) so the api-gateway resolver can
// stash it in LoadMeta, but never the denormalized owner_account_* fields
// (those are filled by the Account loader when ?include[]=owner_account).
func sandboxToProtoFlat(s *domain.SandboxAccount) *pb.SandboxInfo {
	ownID := s.OwnerAccountID
	return &pb.SandboxInfo{
		Id:             s.TypeID,
		Name:           s.Name,
		CreatedAt:      timestamppb.New(s.CreatedAt),
		UpdatedAt:      timestamppb.New(s.UpdatedAt),
		OwnerAccountId: &ownID,
	}
}

func (h *gRPCHandler) DeleteSandbox(ctx context.Context, req *pb.DeleteSandboxRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.sandboxSvc.DeleteSandbox(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) CreateSandbox(ctx context.Context, req *pb.CreateSandboxRequest) (*pb.CreateSandboxResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var mode constants.SandboxMode
	switch req.Mode {
	case pb.SandboxMode_SANDBOX_MODE_SEEDED:
		mode = constants.SandboxModeSeeded
	default:
		mode = constants.SandboxModeBlank
	}

	sandbox, apiErr := h.sandboxSvc.CreateSandbox(ctx, req.Name, mode)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSandboxResponse{
		Sandbox: sandboxToProto(sandbox),
	}, nil
}

func sandboxToProto(s *domain.SandboxAccount) *pb.SandboxInfo {
	info := &pb.SandboxInfo{
		Id:        s.TypeID,
		Name:      s.Name,
		CreatedAt: timestamppb.New(s.CreatedAt),
		UpdatedAt: timestamppb.New(s.UpdatedAt),
	}
	if s.OwnerAccountName != nil {
		ownID := s.OwnerAccountID
		info.OwnerAccountId = &ownID
		info.OwnerAccountName = s.OwnerAccountName
	}
	if s.OwnerAccountCreatedAt != nil {
		info.OwnerAccountCreatedAt = timestamppb.New(*s.OwnerAccountCreatedAt)
	}
	if s.OwnerAccountUpdatedAt != nil {
		info.OwnerAccountUpdatedAt = timestamppb.New(*s.OwnerAccountUpdatedAt)
	}
	return info
}

func (h *gRPCHandler) UpdateAccountSubscription(ctx context.Context, req *pb.UpdateAccountSubscriptionRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var periodEnd *time.Time
	if req.CurrentPeriodEnd != nil {
		t := req.CurrentPeriodEnd.AsTime()
		periodEnd = &t
	}

	apiErr := h.accountSvc.UpdateAccountSubscription(ctx, req.AccountId, req.SubscriptionStatus, req.PlanCode, req.StripeSubscriptionId, periodEnd, req.StripeCustomerId, req.BillingProfileId, req.BillingCadenceId, req.PricingPlanSubscriptionId, req.ServicingStatus, req.CollectionStatus)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ClearAccountStripeCustomer(ctx context.Context, req *pb.ClearAccountStripeCustomerRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.accountSvc.ClearAccountStripeCustomer(ctx, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) UpdateAgentSpendingCap(ctx context.Context, req *pb.UpdateAgentSpendingCapRequest) (*pb.UpdateAgentSpendingCapResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	capCents, apiErr := h.accountSvc.UpdateAgentSpendingCap(ctx, req.CapCents)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAgentSpendingCapResponse{
		CapCents: capCents,
	}, nil
}

func (h *gRPCHandler) GetAccountByStripeCustomerID(ctx context.Context, req *pb.GetAccountByStripeCustomerIDRequest) (*pb.GetAccountByStripeCustomerIDResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	accountID, planCode, apiErr := h.accountSvc.GetAccountByStripeCustomerID(ctx, req.StripeCustomerId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountByStripeCustomerIDResponse{
		AccountId: accountID,
		PlanCode:  planCode,
	}, nil
}

func (h *gRPCHandler) CompleteRegistration(ctx context.Context, req *pb.CompleteRegistrationRequest) (*pb.CompleteRegistrationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	input := domain.CompleteRegistrationInput{
		UserID:           req.UserId,
		PlanCode:         req.PlanCode,
		StripeCustomerID: req.StripeCustomerId,
	}

	if req.StripeSubscriptionId != nil {
		input.StripeSubscriptionID = *req.StripeSubscriptionId
	}

	if req.AccountData != nil {
		input.AccountData = domain.RegistrationAccountData{
			AccountName: req.AccountData.AccountName,
		}

		if req.AccountData.BusinessAddress != nil {
			addr := req.AccountData.BusinessAddress
			input.BusinessAddress = &domain.RegistrationAddress{
				Line1:      derefStr(addr.Line1),
				Line2:      derefStr(addr.Line2),
				City:       derefStr(addr.City),
				State:      derefStr(addr.State),
				PostalCode: derefStr(addr.PostalCode),
				Country:    derefStr(addr.Country),
			}
		}
	}

	result, apiErr := h.accountSvc.CompleteRegistration(ctx, input)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CompleteRegistrationResponse{
		AccountId: result.AccountID,
		SandboxId: result.SandboxID,
	}, nil
}

func unitToProto(u *domain.Unit) *pb.UnitInfo {
	return &pb.UnitInfo{
		Id:                u.ID,
		Name:              u.Name,
		Abbreviation:      u.Abbreviation,
		Type:              u.UnitDimensionCode,
		RatioNumerator:    u.RatioNumerator,
		RatioDenominator:  u.RatioDenominator,
		OffsetNumerator:   u.OffsetNumerator,
		OffsetDenominator: u.OffsetDenominator,
		IsBaseUnit:        u.IsBaseUnit,
		IsInternal:        u.AccountID != nil,
		CreatedAt:         timestamppb.New(u.CreatedAt),
		UpdatedAt:         timestamppb.New(u.UpdatedAt),
		AccountId:         u.AccountID,
	}
}

func (h *gRPCHandler) ListUnits(ctx context.Context, req *pb.ListUnitsRequest) (*pb.ListUnitsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListUnitsParams{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		Query:        req.Query,
		Type:         req.Type,
		UnitGroupIDs: req.UnitGroupIds,
	}

	result, apiErr := h.unitSvc.ListUnits(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbUnits := make([]*pb.UnitInfo, len(result.Units))
	for i, u := range result.Units {
		pbUnits[i] = unitToProto(u)
	}

	return &pb.ListUnitsResponse{
		Units: pbUnits,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetUnit(ctx context.Context, req *pb.GetUnitRequest) (*pb.GetUnitResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	unit, apiErr := h.unitSvc.GetUnit(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetUnitResponse{
		Unit: unitToProto(unit),
	}, nil
}

func (h *gRPCHandler) CreateUnit(ctx context.Context, req *pb.CreateUnitRequest) (*pb.CreateUnitResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateUnitParams{
		Name:              req.Name,
		Abbreviation:      req.Abbreviation,
		UnitDimensionCode: req.Type,
		RatioNumerator:    req.RatioNumerator,
		RatioDenominator:  req.RatioDenominator,
		OffsetNumerator:   req.OffsetNumerator,
		OffsetDenominator: req.OffsetDenominator,
		IsBaseUnit:        req.IsBaseUnit,
	}

	unit, apiErr := h.unitSvc.CreateUnit(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateUnitResponse{
		Unit: unitToProto(unit),
	}, nil
}

func (h *gRPCHandler) UpdateUnit(ctx context.Context, req *pb.UpdateUnitRequest) (*pb.UpdateUnitResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateUnitParams{
		UnitID:            req.Id,
		Name:              req.Name,
		Abbreviation:      req.Abbreviation,
		RatioNumerator:    req.RatioNumerator,
		RatioDenominator:  req.RatioDenominator,
		OffsetNumerator:   req.OffsetNumerator,
		OffsetDenominator: req.OffsetDenominator,
	}

	unit, apiErr := h.unitSvc.UpdateUnit(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateUnitResponse{
		Unit: unitToProto(unit),
	}, nil
}

func (h *gRPCHandler) DeleteUnit(ctx context.Context, req *pb.DeleteUnitRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.unitSvc.DeleteUnit(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) ValidateUnits(ctx context.Context, req *pb.ValidateUnitsRequest) (*pb.ValidateUnitsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ValidateUnitsParams{
		UnitMap: req.UnitMap,
	}

	result, apiErr := h.unitSvc.ValidateUnits(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbUnits := make(map[string]*pb.UnitInfo, len(result.Units))
	for k, u := range result.Units {
		pbUnits[k] = unitToProto(u)
	}

	return &pb.ValidateUnitsResponse{
		Units: pbUnits,
	}, nil
}

func (h *gRPCHandler) BatchGetUnitsByIDs(ctx context.Context, req *pb.BatchGetUnitsByIDsRequest) (*pb.BatchGetUnitsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	units, apiErr := h.unitSvc.BatchGetUnitsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbUnits := make([]*pb.UnitInfo, len(units))
	for i, u := range units {
		pbUnits[i] = unitToProto(u)
	}
	return &pb.BatchGetUnitsByIDsResponse{Units: pbUnits}, nil
}

func (h *gRPCHandler) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	products, apiErr := h.productSvc.SearchProducts(ctx, req.AccountId, req.Query)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbProducts := make([]*pb.ProductInfo, len(products))
	for i, p := range products {
		pbProducts[i] = &pb.ProductInfo{
			ProductId:   p.ProductID,
			ItemId:      p.ItemID,
			Sku:         p.SKU,
			Description: p.Description,
			UnitPrice:   p.UnitPrice,
		}
	}

	return &pb.SearchProductsResponse{Products: pbProducts}, nil
}

func (h *gRPCHandler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	products, apiErr := h.productSvc.ListProducts(ctx, req.AccountId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbProducts := make([]*pb.ProductInfo, len(products))
	for i, p := range products {
		pbProducts[i] = &pb.ProductInfo{
			ProductId:   p.ProductID,
			ItemId:      p.ItemID,
			Sku:         p.SKU,
			Description: p.Description,
			UnitPrice:   p.UnitPrice,
		}
	}

	return &pb.ListProductsResponse{Products: pbProducts}, nil
}

func (h *gRPCHandler) GetCustomerByEmail(ctx context.Context, req *pb.GetCustomerByEmailRequest) (*pb.GetCustomerByEmailResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	customer, apiErr := h.productSvc.GetCustomerByEmail(ctx, req.OwnerAccountId, req.Email)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	if customer == nil {
		return &pb.GetCustomerByEmailResponse{Found: false}, nil
	}

	return &pb.GetCustomerByEmailResponse{
		Found: true,
		Customer: &pb.CustomerInfo{
			RelationId:            customer.RelationID,
			OwnerAccountId:        customer.OwnerAccountID,
			CounterpartyAccountId: customer.CounterpartyAccountID,
			RoleCode:              customer.RoleCode,
			Alias:                 customer.Alias,
			Email:                 customer.Email,
			UserName:              customer.UserName,
		},
	}, nil
}

func paymentTermToProto(pt *domain.PaymentTerm) *pb.PaymentTermInfo {
	return &pb.PaymentTermInfo{
		Id:        pt.ID,
		Name:      pt.Name,
		Status:    string(pt.Status),
		CreatedAt: timestamppb.New(pt.CreatedAt),
		UpdatedAt: timestamppb.New(pt.UpdatedAt),
		AccountId: pt.AccountID,
	}
}

func (h *gRPCHandler) ListPaymentTerms(ctx context.Context, req *pb.ListPaymentTermsRequest) (*pb.ListPaymentTermsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListPaymentTermsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.paymentTermSvc.ListPaymentTerms(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbPaymentTerms := make([]*pb.PaymentTermInfo, len(result.PaymentTerms))
	for i, pt := range result.PaymentTerms {
		pbPaymentTerms[i] = paymentTermToProto(pt)
	}

	return &pb.ListPaymentTermsResponse{
		PaymentTerms: pbPaymentTerms,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetPaymentTerm(ctx context.Context, req *pb.GetPaymentTermRequest) (*pb.GetPaymentTermResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	paymentTerm, apiErr := h.paymentTermSvc.GetPaymentTerm(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetPaymentTermResponse{
		PaymentTerm: paymentTermToProto(paymentTerm),
	}, nil
}

func (h *gRPCHandler) BatchGetPaymentTermsByIDs(ctx context.Context, req *pb.BatchGetPaymentTermsByIDsRequest) (*pb.BatchGetPaymentTermsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	paymentTerms, apiErr := h.paymentTermSvc.BatchGetPaymentTermsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbTerms := make([]*pb.PaymentTermInfo, len(paymentTerms))
	for i, pt := range paymentTerms {
		pbTerms[i] = paymentTermToProto(pt)
	}
	return &pb.BatchGetPaymentTermsByIDsResponse{PaymentTerms: pbTerms}, nil
}

func (h *gRPCHandler) CreatePaymentTerm(ctx context.Context, req *pb.CreatePaymentTermRequest) (*pb.CreatePaymentTermResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreatePaymentTermParams{
		Name: req.Name,
	}

	paymentTerm, apiErr := h.paymentTermSvc.CreatePaymentTerm(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreatePaymentTermResponse{
		PaymentTerm: paymentTermToProto(paymentTerm),
	}, nil
}

func (h *gRPCHandler) UpdatePaymentTerm(ctx context.Context, req *pb.UpdatePaymentTermRequest) (*pb.UpdatePaymentTermResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdatePaymentTermParams{
		PaymentTermID: req.Id,
		Name:          req.Name,
	}

	paymentTerm, apiErr := h.paymentTermSvc.UpdatePaymentTerm(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdatePaymentTermResponse{
		PaymentTerm: paymentTermToProto(paymentTerm),
	}, nil
}

func (h *gRPCHandler) DeletePaymentTerm(ctx context.Context, req *pb.DeletePaymentTermRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.paymentTermSvc.DeletePaymentTerm(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func quantityToProto(q *domain.Quantity) *pb.QuantityInfo {
	if q == nil {
		return nil
	}
	info := &pb.QuantityInfo{
		Id:               q.ID,
		Value:            q.Value,
		UnitId:           q.UnitID,
		UnitAbbreviation: q.UnitAbbreviation,
		UnitType:         q.UnitType,
		UnitName:         q.UnitName,
		CreatedAt:        timestamppb.New(q.CreatedAt),
		UpdatedAt:        timestamppb.New(q.UpdatedAt),
	}
	if q.EmbeddedUnit != nil {
		info.UnitDetail = unitToProto(q.EmbeddedUnit)
	}
	return info
}

func shippingTermToProto(st *domain.ShippingTerm) *pb.ShippingTermInfo {
	var pbLevels []*pb.ServiceLevelInfo
	if len(st.FreeShippingServiceLevels) > 0 {
		pbLevels = make([]*pb.ServiceLevelInfo, len(st.FreeShippingServiceLevels))
		for i, sl := range st.FreeShippingServiceLevels {
			pbLevels[i] = serviceLevelToProto(sl)
		}
	}
	return &pb.ShippingTermInfo{
		Id:                        st.ID,
		Name:                      st.Name,
		Type:                      string(st.Type),
		FlatRate:                  quantityToProto(st.FlatRate),
		MinimumOrderValue:         quantityToProto(st.MinimumOrderValue),
		FreeShippingServiceLevels: pbLevels,
		CreatedAt:                 timestamppb.New(st.CreatedAt),
		UpdatedAt:                 timestamppb.New(st.UpdatedAt),
		AccountId:                 st.AccountID,
	}
}

func (h *gRPCHandler) ListShippingTerms(ctx context.Context, req *pb.ListShippingTermsRequest) (*pb.ListShippingTermsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListShippingTermsParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: req.Includes,
	}

	result, apiErr := h.shippingTermSvc.ListShippingTerms(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbShippingTerms := make([]*pb.ShippingTermInfo, len(result.ShippingTerms))
	for i, st := range result.ShippingTerms {
		pbShippingTerms[i] = shippingTermToProto(st)
	}

	return &pb.ListShippingTermsResponse{
		ShippingTerms: pbShippingTerms,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetShippingTerm(ctx context.Context, req *pb.GetShippingTermRequest) (*pb.GetShippingTermResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	shippingTerm, apiErr := h.shippingTermSvc.GetShippingTerm(ctx, domain.GetShippingTermParams{
		ShippingTermID: req.Id,
		Includes:       req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetShippingTermResponse{
		ShippingTerm: shippingTermToProto(shippingTerm),
	}, nil
}

func protoQuantityInputToDomain(input *pb.QuantityInput) *domain.QuantityInput {
	if input == nil {
		return nil
	}
	return &domain.QuantityInput{
		Value:  input.Value,
		UnitID: input.UnitId,
	}
}

func (h *gRPCHandler) CreateShippingTerm(ctx context.Context, req *pb.CreateShippingTermRequest) (*pb.CreateShippingTermResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateShippingTermParams{
		Name:                        req.Name,
		Type:                        constants.ShippingTermType(req.Type),
		FlatRate:                    protoQuantityInputToDomain(req.FlatRate),
		MinimumOrderValue:           protoQuantityInputToDomain(req.MinimumOrderValue),
		FreeShippingServiceLevelIDs: req.FreeShippingServiceLevelIds,
		Includes:                    req.Includes,
	}

	shippingTerm, apiErr := h.shippingTermSvc.CreateShippingTerm(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateShippingTermResponse{
		ShippingTerm: shippingTermToProto(shippingTerm),
	}, nil
}

func (h *gRPCHandler) UpdateShippingTerm(ctx context.Context, req *pb.UpdateShippingTermRequest) (*pb.UpdateShippingTermResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateShippingTermParams{
		ShippingTermID:              req.Id,
		Name:                        req.Name,
		FlatRate:                    quantityPatchToDomain(req.FlatRate),
		MinimumOrderValue:           quantityPatchToDomain(req.MinimumOrderValue),
		FreeShippingServiceLevelIDs: stringListPatchToSliceField(req.FreeShippingServiceLevelIds),
		Includes:                    req.Includes,
	}
	if req.Type != nil {
		t := constants.ShippingTermType(*req.Type)
		params.Type = &t
	}

	shippingTerm, apiErr := h.shippingTermSvc.UpdateShippingTerm(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateShippingTermResponse{
		ShippingTerm: shippingTermToProto(shippingTerm),
	}, nil
}

func (h *gRPCHandler) DeleteShippingTerm(ctx context.Context, req *pb.DeleteShippingTermRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	apiErr := h.shippingTermSvc.DeleteShippingTerm(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) BatchGetShippingTermsByIDs(ctx context.Context, req *pb.BatchGetShippingTermsByIDsRequest) (*pb.BatchGetShippingTermsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	shippingTerms, apiErr := h.shippingTermSvc.BatchGetShippingTermsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbTerms := make([]*pb.ShippingTermInfo, len(shippingTerms))
	for i, st := range shippingTerms {
		pbTerms[i] = shippingTermToProto(st)
	}
	return &pb.BatchGetShippingTermsByIDsResponse{ShippingTerms: pbTerms}, nil
}

func accountStatusToProto(as *domain.AccountStatus) *pb.AccountStatusInfo {
	return &pb.AccountStatusInfo{
		Id:        as.ID,
		Code:      as.Code,
		Name:      as.Name,
		CreatedAt: timestamppb.New(as.CreatedAt),
		UpdatedAt: timestamppb.New(as.UpdatedAt),
	}
}

func (h *gRPCHandler) ListAccountStatuses(ctx context.Context, req *pb.ListAccountStatusesRequest) (*pb.ListAccountStatusesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAccountStatusesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.accountStatusSvc.ListAccountStatuses(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbStatuses := make([]*pb.AccountStatusInfo, len(result.AccountStatuses))
	for i, as := range result.AccountStatuses {
		pbStatuses[i] = accountStatusToProto(as)
	}

	return &pb.ListAccountStatusesResponse{
		AccountStatuses: pbStatuses,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetAccountStatus(ctx context.Context, req *pb.GetAccountStatusRequest) (*pb.GetAccountStatusResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	accountStatus, apiErr := h.accountStatusSvc.GetAccountStatus(ctx, req.Identifier)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountStatusResponse{
		AccountStatus: accountStatusToProto(accountStatus),
	}, nil
}

func (h *gRPCHandler) BatchGetAccountStatusesByIDs(ctx context.Context, req *pb.BatchGetAccountStatusesByIDsRequest) (*pb.BatchGetAccountStatusesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	statuses, apiErr := h.accountStatusSvc.BatchGetAccountStatusesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbStatuses := make([]*pb.AccountStatusInfo, len(statuses))
	for i, as := range statuses {
		pbStatuses[i] = accountStatusToProto(as)
	}
	return &pb.BatchGetAccountStatusesByIDsResponse{AccountStatuses: pbStatuses}, nil
}

func priorityToProto(p *domain.Priority) *pb.PriorityInfo {
	return &pb.PriorityInfo{
		Id:        p.ID,
		Name:      p.Name,
		Code:      string(p.Code),
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func (h *gRPCHandler) ListPriorities(ctx context.Context, req *pb.ListPrioritiesRequest) (*pb.ListPrioritiesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListPrioritiesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.prioritySvc.ListPriorities(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbPriorities := make([]*pb.PriorityInfo, len(result.Priorities))
	for i, p := range result.Priorities {
		pbPriorities[i] = priorityToProto(p)
	}

	return &pb.ListPrioritiesResponse{
		Priorities: pbPriorities,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetPriority(ctx context.Context, req *pb.GetPriorityRequest) (*pb.GetPriorityResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	priority, apiErr := h.prioritySvc.GetPriority(ctx, req.Identifier)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetPriorityResponse{
		Priority: priorityToProto(priority),
	}, nil
}

func (h *gRPCHandler) BatchGetPrioritiesByIDs(ctx context.Context, req *pb.BatchGetPrioritiesByIDsRequest) (*pb.BatchGetPrioritiesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	priorities, apiErr := h.prioritySvc.BatchGetPrioritiesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbPriorities := make([]*pb.PriorityInfo, len(priorities))
	for i, p := range priorities {
		pbPriorities[i] = priorityToProto(p)
	}
	return &pb.BatchGetPrioritiesByIDsResponse{Priorities: pbPriorities}, nil
}

func accountGroupToProto(ag *domain.AccountGroup) *pb.AccountGroupInfo {
	info := &pb.AccountGroupInfo{
		Id:                 ag.ID,
		Name:               ag.Name,
		CommissionPolicy:   ag.CommissionPolicyCode,
		FreightPolicy:      ag.FreightPolicyCode,
		Type:               ag.AccountGroupTypeCode,
		RegistrationFlowId: ag.RegistrationFlowID,
		CreatedAt:          timestamppb.New(ag.CreatedAt),
		UpdatedAt:          timestamppb.New(ag.UpdatedAt),
	}
	if ag.Description != nil {
		info.Description = ag.Description
	}
	return info
}

func (h *gRPCHandler) ListAccountGroups(ctx context.Context, req *pb.ListAccountGroupsRequest) (*pb.ListAccountGroupsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAccountGroupsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
		Type:   req.Type,
	}

	result, apiErr := h.accountGroupSvc.ListAccountGroups(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbGroups := make([]*pb.AccountGroupInfo, len(result.AccountGroups))
	for i, ag := range result.AccountGroups {
		pbGroups[i] = accountGroupToProto(ag)
	}

	return &pb.ListAccountGroupsResponse{
		AccountGroups: pbGroups,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetAccountGroup(ctx context.Context, req *pb.GetAccountGroupRequest) (*pb.GetAccountGroupResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	accountGroup, apiErr := h.accountGroupSvc.GetAccountGroup(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountGroupResponse{
		AccountGroup: accountGroupToProto(accountGroup),
	}, nil
}

func (h *gRPCHandler) BatchGetAccountGroupsByIDs(ctx context.Context, req *pb.BatchGetAccountGroupsByIDsRequest) (*pb.BatchGetAccountGroupsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	accountGroups, apiErr := h.accountGroupSvc.BatchGetAccountGroupsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbGroups := make([]*pb.AccountGroupInfo, len(accountGroups))
	for i, ag := range accountGroups {
		pbGroups[i] = accountGroupToProto(ag)
	}
	return &pb.BatchGetAccountGroupsByIDsResponse{AccountGroups: pbGroups}, nil
}

func (h *gRPCHandler) CreateAccountGroup(ctx context.Context, req *pb.CreateAccountGroupRequest) (*pb.CreateAccountGroupResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateAccountGroupParams{
		Name:                 req.Name,
		Description:          req.Description,
		AccountGroupTypeCode: req.Type,
		CommissionPolicyCode: req.CommissionPolicy,
		FreightPolicyCode:    req.FreightPolicy,
	}

	accountGroup, apiErr := h.accountGroupSvc.CreateAccountGroup(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAccountGroupResponse{
		AccountGroup: accountGroupToProto(accountGroup),
	}, nil
}

func (h *gRPCHandler) UpdateAccountGroup(ctx context.Context, req *pb.UpdateAccountGroupRequest) (*pb.UpdateAccountGroupResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateAccountGroupParams{
		AccountGroupID:       req.Id,
		Name:                 req.Name,
		Description:          patch.StringFieldFromProto(req.Description),
		CommissionPolicyCode: req.CommissionPolicy,
		FreightPolicyCode:    req.FreightPolicy,
	}

	accountGroup, apiErr := h.accountGroupSvc.UpdateAccountGroup(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAccountGroupResponse{
		AccountGroup: accountGroupToProto(accountGroup),
	}, nil
}

func (h *gRPCHandler) DeleteAccountGroup(ctx context.Context, req *pb.DeleteAccountGroupRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountGroupSvc.DeleteAccountGroup(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func geolocationToProto(g *domain.Geolocation) *pb.GeolocationInfo {
	if g == nil {
		return nil
	}
	info := &pb.GeolocationInfo{
		Id:      g.ID,
		Country: g.Country,
	}
	if g.StreetLine1 != nil {
		info.StreetLine_1 = g.StreetLine1
	}
	if g.StreetLine2 != nil {
		info.StreetLine_2 = g.StreetLine2
	}
	if g.Locality != nil {
		info.Locality = g.Locality
	}
	if g.State != nil {
		info.State = g.State
	}
	if g.PostalCode != nil {
		info.PostalCode = g.PostalCode
	}
	if g.GooglePlaceID != nil {
		info.GooglePlaceId = g.GooglePlaceID
	}
	if g.Latitude != nil {
		info.Latitude = g.Latitude
	}
	if g.Longitude != nil {
		info.Longitude = g.Longitude
	}
	return info
}

func addressToProto(a *domain.Address) *pb.AddressInfo {
	if a == nil {
		return nil
	}
	return &pb.AddressInfo{
		Id:          a.ID,
		Name:        a.Name,
		Phone:       a.Phone,
		Email:       a.Email,
		IsDropShip:  a.IsDropShip,
		Geolocation: geolocationToProto(a.Geolocation),
		CreatedAt:   timestamppb.New(a.CreatedAt),
		UpdatedAt:   timestamppb.New(a.UpdatedAt),
	}
}

func addressComponentsToProto(c *domain.AddressComponents) *pb.AddressComponentsInfo {
	if c == nil {
		return nil
	}
	return &pb.AddressComponentsInfo{
		AddressLine_1: c.AddressLine1,
		AddressLine_2: c.AddressLine2,
		City:          c.City,
		State:         c.State,
		PostalCode:    c.PostalCode,
		Country:       c.Country,
		CountryCode:   c.CountryCode,
	}
}

func (h *gRPCHandler) GetAddress(ctx context.Context, req *pb.GetAddressRequest) (*pb.GetAddressResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	address, apiErr := h.addressSvc.GetAddress(ctx, domain.GetAddressParams{
		AddressID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAddressResponse{
		Address: addressToProto(address),
	}, nil
}

func (h *gRPCHandler) BatchGetAddressesByIDs(ctx context.Context, req *pb.BatchGetAddressesByIDsRequest) (*pb.BatchGetAddressesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	addresses, apiErr := h.addressSvc.BatchGetAddressesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbAddresses := make([]*pb.AddressInfo, len(addresses))
	for i, a := range addresses {
		pbAddresses[i] = addressToProto(a)
	}
	return &pb.BatchGetAddressesByIDsResponse{Addresses: pbAddresses}, nil
}

func (h *gRPCHandler) ListAddresses(ctx context.Context, req *pb.ListAddressesRequest) (*pb.ListAddressesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAddressesParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		DropShip: req.DropShip,
	}

	result, apiErr := h.addressSvc.ListAddresses(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbAddresses := make([]*pb.AddressInfo, len(result.Addresses))
	for i, a := range result.Addresses {
		pbAddresses[i] = addressToProto(a)
	}

	return &pb.ListAddressesResponse{
		Addresses: pbAddresses,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) CreateAddress(ctx context.Context, req *pb.CreateAddressRequest) (*pb.CreateAddressResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateAddressParams{
		Name:        req.Name,
		Phone:       req.Phone,
		Email:       req.Email,
		IsDropShip:  req.IsDropShip,
		StreetLine1: req.StreetLine_1,
		StreetLine2: req.StreetLine_2,
		Locality:    req.Locality,
		State:       req.State,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
	}

	address, apiErr := h.addressSvc.CreateAddress(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAddressResponse{
		Address: addressToProto(address),
	}, nil
}

func (h *gRPCHandler) UpdateAddress(ctx context.Context, req *pb.UpdateAddressRequest) (*pb.UpdateAddressResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateAddressParams{
		AddressID:   req.Id,
		Name:        req.Name,
		Phone:       patch.StringFieldFromProto(req.Phone),
		Email:       patch.StringFieldFromProto(req.Email),
		IsDropShip:  req.IsDropShip,
		StreetLine1: req.StreetLine_1,
		StreetLine2: patch.StringFieldFromProto(req.StreetLine_2),
		Locality:    req.Locality,
		State:       req.State,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
	}

	address, apiErr := h.addressSvc.UpdateAddress(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAddressResponse{
		Address: addressToProto(address),
	}, nil
}

func (h *gRPCHandler) DeleteAddress(ctx context.Context, req *pb.DeleteAddressRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.addressSvc.DeleteAddress(ctx, domain.DeleteAddressParams{
		AddressID: req.Id,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

func (h *gRPCHandler) AutocompleteAddress(ctx context.Context, req *pb.AutocompleteAddressRequest) (*pb.AutocompleteAddressResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	suggestions, apiErr := h.addressValidationSvc.Autocomplete(ctx, req.Input, req.SessionToken)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbSuggestions := make([]*pb.AddressSuggestion, len(suggestions))
	for i, s := range suggestions {
		pbSuggestions[i] = &pb.AddressSuggestion{
			Id:            s.ID,
			Description:   s.Description,
			MainText:      s.MainText,
			SecondaryText: s.SecondaryText,
		}
	}

	return &pb.AutocompleteAddressResponse{
		Suggestions: pbSuggestions,
	}, nil
}

func (h *gRPCHandler) GetAddressDetails(ctx context.Context, req *pb.GetAddressDetailsRequest) (*pb.GetAddressDetailsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.addressValidationSvc.GetPlaceDetails(ctx, req.PlaceId, req.SessionToken)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAddressDetailsResponse{
		Address:          addressComponentsToProto(result.Address),
		FormattedAddress: result.FormattedAddress,
	}, nil
}

func (h *gRPCHandler) ValidateAddress(ctx context.Context, req *pb.ValidateAddressRequest) (*pb.ValidateAddressResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.addressValidationSvc.ValidateAddress(ctx, req.AddressLine_1, req.AddressLine_2, req.City, req.State, req.PostalCode, req.Country)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.ValidateAddressResponse{
		IsValid:            result.IsValid,
		FormattedAddress:   result.FormattedAddress,
		Components:         addressComponentsToProto(result.Components),
		ValidationMessages: result.ValidationMessages,
	}

	return resp, nil
}

func (h *gRPCHandler) ListAccountIntegrations(ctx context.Context, req *pb.ListAccountIntegrationsRequest) (*pb.ListAccountIntegrationsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAccountIntegrationsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.accountIntegrationSvc.ListAccountIntegrations(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbIntegrations := make([]*pb.AccountIntegrationInfo, len(result.AccountIntegrations))
	for i, ai := range result.AccountIntegrations {
		pbIntegrations[i] = accountIntegrationToProto(ai)
	}

	return &pb.ListAccountIntegrationsResponse{
		AccountIntegrations: pbIntegrations,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) BatchGetAccountIntegrationsByIDs(ctx context.Context, req *pb.BatchGetAccountIntegrationsByIDsRequest) (*pb.BatchGetAccountIntegrationsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	integrations, apiErr := h.accountIntegrationSvc.BatchGetAccountIntegrationsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbIntegrations := make([]*pb.AccountIntegrationInfo, len(integrations))
	for i, ai := range integrations {
		pbIntegrations[i] = accountIntegrationToProto(ai)
	}
	return &pb.BatchGetAccountIntegrationsByIDsResponse{AccountIntegrations: pbIntegrations}, nil
}

func (h *gRPCHandler) CreateAccountIntegration(ctx context.Context, req *pb.CreateAccountIntegrationRequest) (*pb.CreateAccountIntegrationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateAccountIntegrationParams{
		Name:            req.Name,
		IntegrationCode: constants.IntegrationCode(req.IntegrationCode),
		Credentials:     req.Credentials,
	}

	integration, apiErr := h.accountIntegrationSvc.CreateAccountIntegration(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAccountIntegrationResponse{
		AccountIntegration: accountIntegrationToProto(integration),
	}, nil
}

func (h *gRPCHandler) UpdateAccountIntegration(ctx context.Context, req *pb.UpdateAccountIntegrationRequest) (*pb.UpdateAccountIntegrationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.UpdateAccountIntegrationParams{
		ID:       req.Id,
		Name:     req.Name,
		IsActive: req.IsActive,
	}

	integration, apiErr := h.accountIntegrationSvc.UpdateAccountIntegration(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAccountIntegrationResponse{
		AccountIntegration: accountIntegrationToProto(integration),
	}, nil
}

func (h *gRPCHandler) DeleteAccountIntegration(ctx context.Context, req *pb.DeleteAccountIntegrationRequest) (*pb.DeleteAccountIntegrationResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.DeleteAccountIntegrationParams{
		ID: req.Id,
	}

	integration, apiErr := h.accountIntegrationSvc.DeleteAccountIntegration(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.DeleteAccountIntegrationResponse{
		AccountIntegration: accountIntegrationToProto(integration),
	}, nil
}

func (h *gRPCHandler) GetStripePublishableKey(ctx context.Context, req *pb.GetStripePublishableKeyRequest) (*pb.GetStripePublishableKeyResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	publishableKey, apiErr := h.accountIntegrationSvc.GetStripePublishableKey(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetStripePublishableKeyResponse{
		PublishableKey: publishableKey,
	}, nil
}

func (h *gRPCHandler) GetStripeStatus(ctx context.Context, req *pb.GetStripeStatusRequest) (*pb.GetStripeStatusResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	hasIntegration, apiErr := h.accountIntegrationSvc.HasStripeIntegration(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetStripeStatusResponse{
		HasStripeIntegration: hasIntegration,
	}, nil
}

func accountIntegrationToProto(ai *domain.AccountIntegration) *pb.AccountIntegrationInfo {
	return &pb.AccountIntegrationInfo{
		Id:              ai.ID,
		Name:            ai.Name,
		IntegrationCode: string(ai.IntegrationCode),
		IsActive:        ai.IsActive,
		CreatedAt:       timestamppb.New(ai.CreatedAt),
		UpdatedAt:       timestamppb.New(ai.UpdatedAt),
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func adjustmentTypeToProto(at *domain.AdjustmentType) *pb.AdjustmentTypeInfo {
	return &pb.AdjustmentTypeInfo{
		Id:        at.ID,
		Name:      at.Name,
		Code:      at.Code,
		CreatedAt: timestamppb.New(at.CreatedAt),
		UpdatedAt: timestamppb.New(at.UpdatedAt),
	}
}

func (h *gRPCHandler) ListAdjustmentTypes(ctx context.Context, req *pb.ListAdjustmentTypesRequest) (*pb.ListAdjustmentTypesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListAdjustmentTypesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.adjustmentTypeSvc.ListAdjustmentTypes(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbAdjustmentTypes := make([]*pb.AdjustmentTypeInfo, len(result.AdjustmentTypes))
	for i, at := range result.AdjustmentTypes {
		pbAdjustmentTypes[i] = adjustmentTypeToProto(at)
	}

	return &pb.ListAdjustmentTypesResponse{
		AdjustmentTypes: pbAdjustmentTypes,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) BatchGetAdjustmentTypesByIDs(ctx context.Context, req *pb.BatchGetAdjustmentTypesByIDsRequest) (*pb.BatchGetAdjustmentTypesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	adjustmentTypes, apiErr := h.adjustmentTypeSvc.BatchGetAdjustmentTypesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbAdjustmentTypes := make([]*pb.AdjustmentTypeInfo, len(adjustmentTypes))
	for i, at := range adjustmentTypes {
		pbAdjustmentTypes[i] = adjustmentTypeToProto(at)
	}
	return &pb.BatchGetAdjustmentTypesByIDsResponse{AdjustmentTypes: pbAdjustmentTypes}, nil
}

func (h *gRPCHandler) BatchGetAccountsByIDs(ctx context.Context, req *pb.BatchGetAccountsByIDsRequest) (*pb.BatchGetAccountsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	accounts, apiErr := h.accountSvc.BatchGetAccountsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbAccounts := make([]*pb.AccountInfo, len(accounts))
	for i, a := range accounts {
		pbAccounts[i] = accountToProto(a)
	}
	return &pb.BatchGetAccountsByIDsResponse{Accounts: pbAccounts}, nil
}

func (h *gRPCHandler) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	account, apiErr := h.accountSvc.GetAccount(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountResponse{
		Account: accountToProto(account),
	}, nil
}

func (h *gRPCHandler) GetAccountBySlug(ctx context.Context, req *pb.GetAccountBySlugRequest) (*pb.GetAccountBySlugResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	account, apiErr := h.accountSvc.GetAccountBySlug(ctx, req.Slug)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountBySlugResponse{
		Account: publicAccountToProto(account),
	}, nil
}

func (h *gRPCHandler) UpdateAccount(ctx context.Context, req *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateAccountParams{
		AccountID:       req.Id,
		Name:            req.Name,
		SupportEmail:    req.SupportEmail,
		PhoneNumber:     req.PhoneNumber,
		Slug:            req.Slug,
		WebsiteURL:      req.WebsiteUrl,
		FacebookHandle:  req.FacebookHandle,
		InstagramHandle: req.InstagramHandle,
		LinkedInHandle:  req.LinkedinHandle,
		TwitterHandle:   req.TwitterHandle,
	}

	account, apiErr := h.accountSvc.UpdateAccount(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateAccountResponse{
		Account: accountToProto(account),
	}, nil
}

func (h *gRPCHandler) UploadAccountPhoto(ctx context.Context, req *pb.UploadAccountPhotoRequest) (*pb.UploadAccountPhotoResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.accountSvc.UploadAccountPhoto(ctx, req.Id, req.File, req.ContentType)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UploadAccountPhotoResponse{
		Success: true,
	}, nil
}

func (h *gRPCHandler) GetAccountLogoURL(ctx context.Context, req *pb.GetAccountLogoURLRequest) (*pb.GetAccountLogoURLResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	url, apiErr := h.accountSvc.GetAccountLogoURL(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAccountLogoURLResponse{
		Url: url,
	}, nil
}

func accountToProto(a *domain.Account) *pb.AccountInfo {
	info := &pb.AccountInfo{
		Id:                       a.ID,
		Name:                     a.Name,
		DefaultBillingAddressId:  a.DefaultBillingAddressID,
		DefaultShippingAddressId: a.DefaultShippingAddressID,
		CreatedAt:                timestamppb.New(a.CreatedAt),
		UpdatedAt:                timestamppb.New(a.UpdatedAt),
	}

	if a.Branding != nil {
		info.Branding = accountBrandingToProto(a.Branding)
	}

	if a.Portal != nil {
		info.Portal = accountPortalToProto(a.Portal)
	}

	return info
}

func accountBrandingToProto(b *domain.AccountBranding) *pb.AccountBrandingInfo {
	return &pb.AccountBrandingInfo{
		Id:              b.ID,
		SupportEmail:    b.SupportEmail,
		PhoneNumber:     b.PhoneNumber,
		LogoUrl:         b.LogoURL,
		FacebookHandle:  b.FacebookHandle,
		InstagramHandle: b.InstagramHandle,
		LinkedinHandle:  b.LinkedInHandle,
		TwitterHandle:   b.TwitterHandle,
		WebsiteUrl:      b.WebsiteURL,
		CreatedAt:       timestamppb.New(b.CreatedAt),
		UpdatedAt:       timestamppb.New(b.UpdatedAt),
	}
}

func accountPortalToProto(p *domain.AccountPortal) *pb.AccountPortalInfo {
	return &pb.AccountPortalInfo{
		Id:        p.ID,
		Slug:      p.Slug,
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func publicAccountToProto(a *domain.PublicAccountBySlug) *pb.PublicAccountInfo {
	return &pb.PublicAccountInfo{
		Id:                      a.ID,
		Name:                    a.Name,
		Slug:                    a.Slug,
		DefaultBillingAddressId: a.DefaultBillingAddressID,
		SupportEmail:            a.SupportEmail,
		LogoUrl:                 a.LogoURL,
	}
}

func permissionToProto(p *domain.Permission) *pb.PermissionInfo {
	return &pb.PermissionInfo{
		Id:                  p.ID,
		Code:                p.Code,
		Name:                p.Name,
		Description:         p.Description,
		PermissionGroupCode: p.PermissionGroupCode,
		CreatedAt:           timestamppb.New(p.CreatedAt),
		UpdatedAt:           timestamppb.New(p.UpdatedAt),
	}
}

func permissionGroupToProto(pg *domain.PermissionGroup) *pb.PermissionGroupInfo {
	perms := make([]*pb.PermissionInfo, len(pg.Permissions))
	for i, p := range pg.Permissions {
		perms[i] = permissionToProto(p)
	}
	return &pb.PermissionGroupInfo{
		Id:          pg.ID,
		Code:        pg.Code,
		Name:        pg.Name,
		Description: pg.Description,
		Permissions: perms,
		CreatedAt:   timestamppb.New(pg.CreatedAt),
		UpdatedAt:   timestamppb.New(pg.UpdatedAt),
	}
}

func (h *gRPCHandler) ListPermissionGroups(ctx context.Context, req *pb.ListPermissionGroupsRequest) (*pb.ListPermissionGroupsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListPermissionGroupsParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.permissionGroupSvc.ListPermissionGroups(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbGroups := make([]*pb.PermissionGroupInfo, len(result.PermissionGroups))
	for i, pg := range result.PermissionGroups {
		pbGroups[i] = permissionGroupToProto(pg)
	}

	return &pb.ListPermissionGroupsResponse{
		PermissionGroups: pbGroups,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) BatchGetPermissionGroupsByIDs(ctx context.Context, req *pb.BatchGetPermissionGroupsByIDsRequest) (*pb.BatchGetPermissionGroupsByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	groups, apiErr := h.permissionGroupSvc.BatchGetPermissionGroupsByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbGroups := make([]*pb.PermissionGroupInfo, len(groups))
	for i, pg := range groups {
		pbGroups[i] = permissionGroupToProto(pg)
	}

	return &pb.BatchGetPermissionGroupsByIDsResponse{
		PermissionGroups: pbGroups,
	}, nil
}

// Product Type handlers

func productTypeToProto(pt *domain.ProductType) *pb.ProductTypeInfo {
	return &pb.ProductTypeInfo{
		Id:        pt.ID,
		Name:      pt.Name,
		Code:      pt.Code,
		CreatedAt: timestamppb.New(pt.CreatedAt),
		UpdatedAt: timestamppb.New(pt.UpdatedAt),
	}
}

func (h *gRPCHandler) ListProductTypes(ctx context.Context, req *pb.ListProductTypesRequest) (*pb.ListProductTypesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListProductTypesParams{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	result, apiErr := h.productTypeSvc.ListProductTypes(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbProductTypes := make([]*pb.ProductTypeInfo, len(result.ProductTypes))
	for i, pt := range result.ProductTypes {
		pbProductTypes[i] = productTypeToProto(pt)
	}

	return &pb.ListProductTypesResponse{
		ProductTypes: pbProductTypes,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetProductType(ctx context.Context, req *pb.GetProductTypeRequest) (*pb.GetProductTypeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	productType, apiErr := h.productTypeSvc.GetProductType(ctx, req.Identifier)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetProductTypeResponse{
		ProductType: productTypeToProto(productType),
	}, nil
}

func (h *gRPCHandler) BatchGetProductTypesByIDs(ctx context.Context, req *pb.BatchGetProductTypesByIDsRequest) (*pb.BatchGetProductTypesByIDsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	productTypes, apiErr := h.productTypeSvc.BatchGetProductTypesByIDs(ctx, req.Ids)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	pbProductTypes := make([]*pb.ProductTypeInfo, len(productTypes))
	for i, pt := range productTypes {
		pbProductTypes[i] = productTypeToProto(pt)
	}
	return &pb.BatchGetProductTypesByIDsResponse{ProductTypes: pbProductTypes}, nil
}

func (h *gRPCHandler) CreateProductType(ctx context.Context, req *pb.CreateProductTypeRequest) (*pb.CreateProductTypeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateProductTypeParams{
		Name: req.Name,
		Code: req.Code,
	}

	productType, apiErr := h.productTypeSvc.CreateProductType(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateProductTypeResponse{
		ProductType: productTypeToProto(productType),
	}, nil
}

func (h *gRPCHandler) UpdateProductType(ctx context.Context, req *pb.UpdateProductTypeRequest) (*pb.UpdateProductTypeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateProductTypeParams{
		ProductTypeID: req.Id,
		Name:          req.Name,
		Code:          req.Code,
	}

	productType, apiErr := h.productTypeSvc.UpdateProductType(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateProductTypeResponse{
		ProductType: productTypeToProto(productType),
	}, nil
}

func (h *gRPCHandler) DeleteProductType(ctx context.Context, req *pb.DeleteProductTypeRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.productTypeSvc.DeleteProductType(ctx, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
