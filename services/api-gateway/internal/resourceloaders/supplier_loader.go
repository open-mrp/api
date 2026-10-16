package resourceloaders

import (
	"context"
	"strings"
	"time"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var supplierLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.supplier")

// Load-meta keys the supplier presenter stashes for the includes that resolve a supplier's related records.
const (
	SupplierMetaCarrierID      = "defaults_carrier_id"
	SupplierMetaServiceLevelID = "defaults_service_level_id"
	SupplierMetaPaymentTermID  = "defaults_payment_term_id"
	SupplierMetaShippingTermID = "defaults_shipping_term_id"
	SupplierMetaBillToAddress  = "defaults_bill_to_address"
	SupplierMetaShipToAddress  = "defaults_ship_to_address"
)

// SupplierCoreIncludes are the includes core-service needs for a supplier request: the default addresses, which
// belong to the supplier's own account and so are joined there, and for preview.9 callers the item count.
func SupplierCoreIncludes(ctx context.Context) []string {
	var out []string
	for _, key := range include.RequestedIncludes(ctx) {
		for prefix, core := range map[string]string{"defaults.bill_to_address": "bill_to_address", "defaults.ship_to_address": "ship_to_address"} {
			if key == prefix || strings.HasPrefix(key, prefix+".") {
				out = append(out, core)
			}
		}
	}
	if IsPreview9(ctx) {
		out = append(out, "material_count")
	}
	return out
}

// LoadSuppliers reads the suppliers documents name, presented in full; their related records load through their
// own includes.
func LoadSuppliers(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, supplierLoaderTracer, "loader.suppliers.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetSuppliersByIDsResponse, error) {
			return coreClient.BatchGetSuppliersByIDs(ctx, &pb.BatchGetSuppliersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]any, len(resp.Suppliers))
	for _, s := range resp.Suppliers {
		out[s.Id] = SupplierFromSummaryProto(ctx, s)
	}
	return out, nil
}

// supplierFields is what both supplier protos carry.
type supplierFields struct {
	id, name, number, status                         string
	instructions, email, phone, url                  *string
	billingType, billingAccount                      *string
	carrierID, serviceLevelID, paymentTermID, termID *string
	materialCount                                    *int64
	billTo, shipTo                                   *pb.CustomerAddressProto
	createdAt, updatedAt                             *timestamppb.Timestamp
}

func presentSupplier(ctx context.Context, f supplierFields) *apiresource.Supplier {
	status := constants.ArchiveStatus(f.status)
	if !status.IsValid() {
		status = constants.ArchiveStatusActive
	}
	billing := constants.CarrierBillingTypeSender
	if f.billingType != nil && *f.billingType != "" {
		billing = constants.CarrierBillingType(*f.billingType)
	}
	s := &apiresource.Supplier{
		ID:             f.id,
		Object:         constants.ObjectTypeSupplier,
		Status:         status,
		Name:           f.name,
		Number:         f.number,
		Instructions:   f.instructions,
		Contact:        &apiresource.SupplierContact{Email: f.email, Phone: f.phone, URL: f.url},
		CarrierBilling: &apiresource.SupplierCarrierBilling{Type: billing, AccountNumber: f.billingAccount},
		Defaults:       &apiresource.SupplierDefaults{},
		MaterialCount:  f.materialCount,
		CreatedAt:      timeOrZero(f.createdAt),
		UpdatedAt:      timeOrZero(f.updatedAt),
	}
	meta := include.GetLoadMeta(ctx)
	set := func(key string, value *string) {
		if value != nil && *value != "" {
			meta.Set(constants.ObjectTypeSupplier, s.ID, key, *value)
		}
	}
	set(SupplierMetaCarrierID, f.carrierID)
	set(SupplierMetaServiceLevelID, f.serviceLevelID)
	set(SupplierMetaPaymentTermID, f.paymentTermID)
	set(SupplierMetaShippingTermID, f.termID)
	if f.billTo != nil {
		meta.Set(constants.ObjectTypeSupplier, s.ID, SupplierMetaBillToAddress, CustomerAddressFromProto(f.billTo))
	}
	if f.shipTo != nil {
		meta.Set(constants.ObjectTypeSupplier, s.ID, SupplierMetaShipToAddress, CustomerAddressFromProto(f.shipTo))
	}
	return s
}

func timeOrZero(t *timestamppb.Timestamp) time.Time {
	if p := grpcutil.TimestampToTimePtr(t); p != nil {
		return *p
	}
	return time.Time{}
}

func materialCountOf(has bool, n int64) *int64 {
	if !has {
		return nil
	}
	return &n
}

// SupplierFromProto presents a supplier read on its own, stashing its related records' IDs for their includes.
func SupplierFromProto(ctx context.Context, s *pb.SupplierProto) *apiresource.Supplier {
	if s == nil {
		return nil
	}
	return presentSupplier(ctx, supplierFields{
		id: s.Id, name: s.Name, number: s.Number, status: s.Status,
		instructions: s.Instructions, email: s.Email, phone: s.Phone, url: s.Url,
		billingType: s.CarrierBillingType, billingAccount: s.CarrierBillingAccount,
		carrierID: s.DefaultCarrierId, serviceLevelID: s.DefaultServiceLevelId, paymentTermID: s.PaymentTermId, termID: s.ShippingTermId,
		materialCount: materialCountOf(s.HasMaterialCount, s.MaterialCount),
		billTo:        s.BillToAddress, shipTo: s.ShipToAddress,
		createdAt: s.CreatedAt, updatedAt: s.UpdatedAt,
	})
}

// SupplierFromSummaryProto presents a supplier read by a list or a batch.
func SupplierFromSummaryProto(ctx context.Context, s *pb.SupplierSummaryProto) *apiresource.Supplier {
	if s == nil {
		return nil
	}
	return presentSupplier(ctx, supplierFields{
		id: s.Id, name: s.Name, number: s.Number, status: s.Status,
		instructions: s.Instructions, email: s.Email, phone: s.Phone, url: s.Url,
		billingType: s.CarrierBillingType, billingAccount: s.CarrierBillingAccount,
		carrierID: s.DefaultCarrierId, serviceLevelID: s.DefaultServiceLevelId, paymentTermID: s.PaymentTermId, termID: s.ShippingTermId,
		materialCount: materialCountOf(s.HasMaterialCount, s.MaterialCount),
		billTo:        s.BillToAddress, shipTo: s.ShipToAddress,
		createdAt: s.CreatedAt, updatedAt: s.UpdatedAt,
	})
}

// SupplierItemFromProto presents a supplier item, stashing its supplier's and item's IDs for their includes.
func SupplierItemFromProto(ctx context.Context, s *pb.SupplierItemInfo) *apiresource.SupplierItem {
	if s == nil {
		return nil
	}
	status := constants.ArchiveStatus(s.Status)
	if !status.IsValid() {
		status = constants.ArchiveStatusActive
	}
	meta := include.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeSupplierItem, s.Id, "supplier_id", s.SupplierId)
	meta.Set(constants.ObjectTypeSupplierItem, s.Id, "item_id", s.ItemId)
	return &apiresource.SupplierItem{
		ID:                  s.Id,
		Object:              constants.ObjectTypeSupplierItem,
		Status:              status,
		SupplierSKU:         s.SupplierSku,
		SupplierDescription: s.SupplierDescription,
		CreatedAt:           timeOrZero(s.CreatedAt),
		UpdatedAt:           timeOrZero(s.UpdatedAt),
	}
}
