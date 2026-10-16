package customerep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to update a customer.
type UpdateCustomerRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
	// The customer's business name, as shown throughout the app and on documents.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Customer number, unique within your account.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Whether the customer can receive new orders: `active` or `archived`.
	Status field.Optional[constants.ArchiveStatus] `json:"status,omitzero"`
	// The customer's credit standing: `normal`, `hold_shipment` or `hold_all`.
	Standing field.Optional[constants.CustomerStanding] `json:"standing,omitzero"`
	// Your team's standing instructions for this customer's orders. Send `null` to clear them.
	Instructions field.Clearable[string] `json:"instructions,omitzero"`
	// Whether orders from this customer earn sales commission: `applied` or `exempt`.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Whether this customer is billed for freight: `billed` or `free`.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// Credit extended to this customer. Send `null` for no limit.
	CreditLimit field.Clearable[apirequest.MoneyInput] `json:"credit_limit,omitzero"`
	// How to reach the customer's business. Only fields you send are changed.
	Contact field.Optional[CustomerContactInput] `json:"contact,omitzero"`
	// Who pays the carrier. Only fields you send are changed.
	CarrierBilling field.Optional[CustomerCarrierBillingInput] `json:"carrier_billing,omitzero"`
	// Values copied onto a new order for this customer. Only fields you send are changed.
	Defaults field.Optional[UpdateCustomerDefaultsInput] `json:"defaults,omitzero"`
	// ID of the account group that classifies this customer.
	GroupID field.Optional[string] `json:"group_id,omitzero"`
	// IDs of the pricing account groups the customer belongs to. Replaces the current set.
	PriceGroupIDs field.Optional[[]string] `json:"price_group_ids,omitzero" visibility:"internal"`
	// IDs of the price lists assigned directly to the customer, at most 100. Replaces the current set; send `[]` for none.
	PriceListIDs field.Optional[[]string] `json:"price_list_ids,omitzero" validate:"omitempty,max=100"`
	// ID of the customer this one belongs to. Send `null` to make it a standalone customer.
	ParentCustomerID field.Clearable[string] `json:"parent_customer_id,omitzero"`
	// Who receives this customer's documents by email. Replaces the current list; send `[]` for none.
	//
	// Each recipient is one of the customer's users and the documents they get: `acknowledgment`, `invoice`, `shipment` or `statement`.
	NotificationRecipients field.Optional[[]apirequest.EmailRecipientInput] `json:"notification_recipients,omitzero"`
}

var sampleUpdateCustomerRequest = &UpdateCustomerRequest{
	Name:     field.Some("Acme Corp"),
	Standing: field.Some(constants.CustomerStandingHoldShipment),
	Contact:  field.Some(CustomerContactInput{Email: field.Set("orders@acme.com")}),
	Defaults: field.Some(UpdateCustomerDefaultsInput{
		CarrierID: field.Some(apiresource.SampleCarrierID),
		LeadTime:  field.Set(apirequest.QuantityInput{Value: "5", UnitID: apirequest.DayUnitID}),
	}),
	NotificationRecipients: field.Some([]apirequest.EmailRecipientInput{{
		AccountUserID: apiresource.SampleAccountUserID,
		Documents:     []constants.RecipientDocument{constants.RecipientDocumentInvoice},
	}}),
}

func (*UpdateCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateCustomerRequest)
}

// Updates a customer.
//
// Only fields you send are changed, including fields inside `contact`, `carrier_billing` and `defaults`. Send `null` to clear a clearable field.
type UpdateCustomerEndpoint struct{}

func (e *UpdateCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateCustomerRequest, *apiresource.Customer] {
	return (&apiendpoint.APIEndpoint[*UpdateCustomerRequest, *apiresource.Customer]{
		Title:             "Update Customer",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeCustomer,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateCustomerRequest) (*apiresource.Customer, *apierror.APIError) {
			return svc.(CustomerSvc).UpdateCustomer
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomer,
			Fields:     customerIncludeFields,
		}),
	})
}
