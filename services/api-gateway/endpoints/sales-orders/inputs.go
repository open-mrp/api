package salesorderep

import (
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

func init() {
	validate.RegisterWrappedTypes(
		field.Optional[CreateSalesOrderFreightInput]{},
		field.Optional[UpdateSalesOrderFreightInput]{},
		field.Optional[CarrierBillingInput]{},
		field.Clearable[CarrierBillingInput]{},
		field.Optional[CreateSalesOrderCommitmentInput]{},
		field.Optional[UpdateSalesOrderCommitmentInput]{},
		field.Optional[[]apirequest.EmailRecipientInput]{},
		field.Optional[[]SalesOrderEmailContactInput]{},
	)
}

// An account user to receive one kind of an order's documents.
type SalesOrderEmailContactInput struct {
	// ID of the customer's account user.
	AccountUserID string `json:"account_user_id" validate:"required"`
}

// emailContactListToProto is a replacement list for one kind of document, or nil when it was left out.
func emailContactListToProto(f field.Optional[[]SalesOrderEmailContactInput]) *pb.SalesOrderEmailContactList {
	contacts, ok := f.Value()
	if !ok {
		return nil
	}
	out := &pb.SalesOrderEmailContactList{Contacts: make([]*pb.SalesOrderEmailContactInput, len(contacts))}
	for i, c := range contacts {
		out.Contacts[i] = &pb.SalesOrderEmailContactInput{AccountUserId: c.AccountUserID}
	}
	return out
}

// salesOrderIncludeFields are the include keys every endpoint returning a sales order accepts.
var salesOrderIncludeFields = []string{
	"customer", "site", "sales_rep", "sales_rep.user", "created_by", "bill_to_address", "ship_to_address", "contacts",
	"bill_to.address", "ship_to.address", "freight.carrier", "freight.service_level",
	"payment_term", "shipping_term", "order_discount", "totals",
	"email_recipients", "email_recipients.account_user",
	"related.pick", "related.jobs", "related.shipments", "related.invoices",
	"lines", "lines.product", "lines.product.item", "lines.product.item.category", "lines.product.item.category.properties",
	"lines.product.item.category.unit_group", "lines.product.item.category.unit_group.base_unit",
	"lines.product.item.category.unit_group.associated_units", "lines.product.item.category.unit_group.associated_units.unit",
	"lines.product.product_line", "lines.quantity_ordered", "lines.unit_price", "lines.unit_cost", "lines.totals",
}

func salesOrderIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeSalesOrder,
		Fields:     salesOrderIncludeFields,
	})
}
