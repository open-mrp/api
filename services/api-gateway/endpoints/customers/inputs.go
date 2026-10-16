package customerep

import (
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

func init() {
	validate.RegisterWrappedTypes(
		field.Optional[CustomerContactInput]{},
		field.Optional[CustomerCarrierBillingInput]{},
		field.Optional[UpdateCustomerDefaultsInput]{},
		field.Optional[[]apirequest.EmailRecipientInput]{},
	)
}

// customerIncludeFields are the include keys every customer endpoint accepts.
var customerIncludeFields = []string{
	"group", "price_groups", "price_lists", "parent_customer", "child_customers",
	"notification_recipients", "notification_recipients.account_user",
	"defaults.carrier", "defaults.carrier.service_levels", "defaults.service_level", "defaults.payment_term",
	"defaults.shipping_term", "defaults.sales_rep", "defaults.sales_rep.user", "defaults.bill_to_address",
	"defaults.ship_to_address", "defaults.receive_calendar",
}

// How to reach a customer's business.
type CustomerContactInput struct {
	// Email address. Send `null` to clear it.
	Email field.Clearable[string] `json:"email,omitzero" validate:"omitempty,max=255"`
	// Phone number. Send `null` to clear it.
	Phone field.Clearable[string] `json:"phone,omitzero" validate:"omitempty,max=255"`
	// Website URL. Send `null` to clear it.
	URL field.Clearable[string] `json:"url,omitzero" validate:"omitempty,max=255"`
}

// Who pays the carrier for a customer's shipments.
type CustomerCarrierBillingInput struct {
	// `sender`: you pay the carrier. `third_party`: the carrier bills `account_number`. Defaults to `sender`.
	Type field.Optional[constants.CarrierBillingType] `json:"type,omitzero"`
	// The carrier account billed when `type` is `third_party`. Send `null` to clear it.
	AccountNumber field.Clearable[string] `json:"account_number,omitzero" validate:"omitempty,max=255"`
}

// Values copied onto a new order for the customer when the order doesn't set them.
type CreateCustomerDefaultsInput struct {
	// ID of the carrier for the customer's orders.
	CarrierID string `json:"carrier_id" validate:"required"`
	// ID of a service level of the carrier.
	ServiceLevelID field.Optional[string] `json:"service_level_id,omitzero"`
	// ID of the payment term for the customer's orders.
	PaymentTermID string `json:"payment_term_id" validate:"required"`
	// ID of the shipping term for the customer's orders.
	ShippingTermID string `json:"shipping_term_id" validate:"required"`
	// ID of your account user credited as the sales rep on the customer's orders.
	SalesRepID field.Optional[string] `json:"sales_rep_id,omitzero"`
	// ID of the receive calendar naming the days the customer's dock accepts freight.
	ReceiveCalendarID field.Optional[string] `json:"receive_calendar_id,omitzero" visibility:"internal"`
	// Business days from an order being issued to it being due to ship, in the `day` unit. Omit to inherit.
	LeadTime field.Optional[apirequest.QuantityInput] `json:"lead_time,omitzero"`
	// Where the customer's order lines are supplied from when the item doesn't say: `stock`, `make` or `buy`.
	Supply field.Optional[constants.Supply] `json:"supply,omitzero"`
}

// The customer's defaults to change. Only fields you send are changed.
type UpdateCustomerDefaultsInput struct {
	// ID of the carrier for the customer's orders.
	CarrierID field.Optional[string] `json:"carrier_id,omitzero"`
	// ID of a service level of the carrier. Send `null` to clear it.
	ServiceLevelID field.Clearable[string] `json:"service_level_id,omitzero"`
	// ID of the payment term for the customer's orders.
	PaymentTermID field.Optional[string] `json:"payment_term_id,omitzero"`
	// ID of the shipping term for the customer's orders.
	ShippingTermID field.Optional[string] `json:"shipping_term_id,omitzero"`
	// ID of your account user credited as the sales rep on the customer's orders. Send `null` to clear it.
	SalesRepID field.Clearable[string] `json:"sales_rep_id,omitzero"`
	// ID of one of the customer's addresses to bill its orders to. Send `null` to clear it.
	BillToAddressID field.Clearable[string] `json:"bill_to_address_id,omitzero"`
	// ID of one of the customer's addresses to ship its orders to. Send `null` to clear it.
	ShipToAddressID field.Clearable[string] `json:"ship_to_address_id,omitzero"`
	// ID of the receive calendar naming the days the customer's dock accepts freight. Send `null` to clear it.
	ReceiveCalendarID field.Clearable[string] `json:"receive_calendar_id,omitzero" visibility:"internal"`
	// Business days from an order being issued to it being due to ship, in the `day` unit. Send `null` to inherit.
	LeadTime field.Clearable[apirequest.QuantityInput] `json:"lead_time,omitzero"`
	// Where the customer's order lines are supplied from when the item doesn't say. Send `null` to use the item's.
	Supply field.Clearable[constants.Supply] `json:"supply,omitzero"`
}

// recipientsToProto is the notification recipients as core-service takes them.
func recipientsToProto(in []apirequest.EmailRecipientInput) []*pb.EmailRecipient {
	return apirequest.EmailRecipientsToProto(in)
}
