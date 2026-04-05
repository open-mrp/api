package receivableep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ReceivableEntryPresenter(e *pb.ReceivableEntryProto) apiresource.ReceivableEntry {
	if e == nil {
		return apiresource.ReceivableEntry{}
	}

	entry := apiresource.ReceivableEntry{
		Object:           constants.ObjectTypeReceivableEntry,
		PONumber:         e.PoNumber,
		InvoicedAt:       grpcutil.TimestampToTime(e.InvoicedAt),
		RemainingBalance: e.RemainingBalance,
		IsPaidInFull:     e.IsPaidInFull,
	}

	entry.Invoice = &apiresource.Invoice{
		ID:     e.InvoiceId,
		Object: constants.ObjectTypeInvoice,
		Number: e.InvoiceNumber,
	}

	entry.Customer = &apiresource.Customer{
		ID:     e.CustomerId,
		Object: constants.ObjectTypeCustomer,
		Name:   e.CustomerName,
		Number: e.CustomerNumber,
	}

	return entry
}

func ReceivableEntryListPresenter(resp interface {
	GetReceivables() []*pb.ReceivableEntryProto
	GetPageInfo() *pb.PageInfo
}) *apiresource.List[apiresource.ReceivableEntry] {
	if resp == nil {
		return apiresource.NewList[apiresource.ReceivableEntry](nil, apiresource.PageInfo{})
	}

	receivables := resp.GetReceivables()
	items := make([]apiresource.ReceivableEntry, len(receivables))
	for i, e := range receivables {
		items[i] = ReceivableEntryPresenter(e)
	}

	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(resp.GetPageInfo()))
}
