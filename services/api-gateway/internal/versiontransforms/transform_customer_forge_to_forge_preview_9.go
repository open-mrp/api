package versiontransforms

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped customers and account groups:
//   - customer: the credit standing moved from status to standing, and status became the lifecycle (active |
//     archived); note became instructions; contact_info became contact; freight_preferences dissolved into a
//     top-level freight_policy, carrier_billing { type, account_number } and defaults.carrier / service_level;
//     bill_to_address and ship_to_address moved under defaults; defaults lost priority, and gained lead_time (a day
//     Quantity, was lead_time_days), receive_calendar (was receive_calendar_id) and supply (was fulfillment_policy);
//     type became group, parent_account parent_customer, child_accounts child_customers; and
//     notification_preferences gave way to notification_recipients. Sections carry no object.
//   - account group: default_lead_time_days became defaults.lead_time.
//   - customer lead time: days became lead_time, and customer an expandable customer instead of an entity.
//
// preview.9 callers get the old shapes back and their writes, list filters and include keys upgraded. preview.9's
// default priority is gone, so it reads as null and writes of it are dropped. A merge runs as a job now; preview.9
// callers get the job too, since a merge can't be waited for here.
type customerForgeToPreview9 struct {
	forgeToPreview9
	everyObject
}

func init() {
	version.RegisterTransformer(customerForgeToPreview9{})
}

const (
	customerRoute       = "/v1/sales/customers/{id}"
	accountGroupsRoute  = "/v1/sales/account-groups"
	accountGroupRoute   = "/v1/sales/account-groups/{id}"
	preview9DayUnitID   = "day"
	preview9MakeToStock = "make_to_stock"
	preview9MakeToOrder = "make_to_order"
)

// ForcedIncludes loads what the downgrade reads: a customer's receive calendar, for receive_calendar_id, and a lead
// time's customer, for its entity.
func (customerForgeToPreview9) ForcedIncludes(objectType object.Type) []string {
	switch objectType {
	case constants.ObjectTypeCustomer:
		return []string{"defaults.receive_calendar"}
	case constants.ObjectTypeCustomerLeadTime:
		return []string{"customer"}
	}
	return nil
}

func (customerForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		switch obj["object"] {
		case string(constants.ObjectTypeCustomer):
			downgradeCustomer(obj)
		case string(constants.ObjectTypeAccountGroup):
			downgradeAccountGroup(obj)
		case string(constants.ObjectTypeCustomerLeadTime):
			downgradeCustomerLeadTime(obj)
		}
	})
	return data
}

func downgradeCustomer(obj map[string]any) {
	if _, forge := obj["standing"]; !forge {
		return
	}
	obj["status"] = obj["standing"]
	delete(obj, "standing")
	renameKey(obj, "instructions", "note")

	contact := map[string]any{"object": string(constants.ObjectTypeCustomerContactInfo), "email": nil, "phone": nil, "url": nil}
	if c, ok := obj["contact"].(map[string]any); ok {
		contact["email"], contact["phone"], contact["url"] = c["email"], c["phone"], c["url"]
	}
	obj["contact_info"] = contact
	delete(obj, "contact")

	defaults, _ := obj["defaults"].(map[string]any)
	if defaults == nil {
		defaults = map[string]any{}
	}
	freight := map[string]any{
		"object":          string(constants.ObjectTypeCustomerFreightPreferences),
		"status":          obj["freight_policy"],
		"carrier":         defaults["carrier"],
		"service_level":   defaults["service_level"],
		"billing_type":    nil,
		"billing_account": nil,
	}
	if billing, ok := obj["carrier_billing"].(map[string]any); ok {
		freight["billing_type"], freight["billing_account"] = billing["type"], billing["account_number"]
	}
	obj["freight_preferences"] = freight
	delete(obj, "freight_policy")
	delete(obj, "carrier_billing")

	obj["bill_to_address"] = defaults["bill_to_address"]
	obj["ship_to_address"] = defaults["ship_to_address"]
	obj["defaults"] = map[string]any{
		"object":              string(constants.ObjectTypeCustomerDefaults),
		"payment_term":        defaults["payment_term"],
		"shipping_term":       defaults["shipping_term"],
		"priority":            nil,
		"sales_rep":           defaults["sales_rep"],
		"lead_time_days":      preview9LeadTimeDays(defaults["lead_time"]),
		"receive_calendar_id": idOf(defaults["receive_calendar"]),
		"fulfillment_policy":  preview9FulfillmentPolicy(defaults["supply"]),
	}

	renameKey(obj, "group", "type")
	renameKey(obj, "parent_customer", "parent_account")
	renameKey(obj, "child_customers", "child_accounts")

	var prefs any
	if recipients, ok := obj["notification_recipients"].(map[string]any); ok {
		prefs = map[string]any{
			"object":                 string(constants.ObjectTypeCustomerNotificationPreferences),
			"accepts_invoice_emails": recipientsGetInvoices(recipients),
		}
	}
	obj["notification_preferences"] = prefs
	delete(obj, "notification_recipients")
}

// recipientsGetInvoices reports whether any of a recipients list's people receive invoices.
func recipientsGetInvoices(list map[string]any) bool {
	data, _ := list["data"].([]any)
	for _, item := range data {
		r, _ := item.(map[string]any)
		docs, _ := r["documents"].([]any)
		for _, d := range docs {
			if d == string(constants.RecipientDocumentInvoice) {
				return true
			}
		}
	}
	return false
}

// preview9LeadTimeDays is a forge day Quantity as preview.9's whole days, or nil.
func preview9LeadTimeDays(v any) any {
	q, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	s, _ := q["value"].(string)
	days, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return days
}

func preview9FulfillmentPolicy(v any) any {
	switch v {
	case string(constants.SupplyStock):
		return preview9MakeToStock
	case string(constants.SupplyMake):
		return preview9MakeToOrder
	}
	return nil
}

func downgradeAccountGroup(obj map[string]any) {
	defaults, ok := obj["defaults"].(map[string]any)
	if !ok {
		return
	}
	obj["default_lead_time_days"] = preview9LeadTimeDays(defaults["lead_time"])
	delete(obj, "defaults")
}

func downgradeCustomerLeadTime(obj map[string]any) {
	if _, forge := obj["lead_time"]; !forge {
		return
	}
	obj["days"] = preview9LeadTimeDays(obj["lead_time"])
	delete(obj, "lead_time")
	var entity any
	if id := idOf(obj["customer"]); id != nil {
		entity = map[string]any{"id": id, "object": string(constants.ObjectTypeEntity), "type": string(constants.ObjectTypeCustomer), "name": nil, "handle": nil}
	}
	obj["customer"] = entity
}

func (customerForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

func (customerForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch {
	case method == http.MethodPost && route == customersRoute,
		method == http.MethodPatch && route == customerRoute:
		upgradeCustomerWrite(data)
	case method == http.MethodPost && route == accountGroupsRoute,
		method == http.MethodPatch && route == accountGroupRoute:
		if days, ok := data["default_lead_time_days"]; ok {
			delete(data, "default_lead_time_days")
			data["defaults"] = map[string]any{"lead_time": forgeLeadTime(days)}
		}
	}
	return data
}

// preview9CustomerDefaults are preview.9's flat customer fields that moved under defaults, by their forge name.
var preview9CustomerDefaults = map[string]string{
	"default_carrier_id":       "carrier_id",
	"default_service_level_id": "service_level_id",
	"default_payment_term_id":  "payment_term_id",
	"default_shipping_term_id": "shipping_term_id",
	"default_sales_rep_id":     "sales_rep_id",
	"bill_to_address_id":       "bill_to_address_id",
	"ship_to_address_id":       "ship_to_address_id",
	"receive_calendar_id":      "receive_calendar_id",
}

// upgradeCustomerWrite moves a preview.9 customer write's flat fields into forge's sections.
func upgradeCustomerWrite(body map[string]any) {
	renameKey(body, "status", "standing")
	renameKey(body, "note", "instructions")
	renameKey(body, "customer_type_group_id", "group_id")
	renameKey(body, "customer_price_group_ids", "price_group_ids")
	delete(body, "default_priority")

	section := func(name string) map[string]any {
		m, _ := body[name].(map[string]any)
		if m == nil {
			m = map[string]any{}
		}
		return m
	}
	move := func(from, to string, into map[string]any) bool {
		v, ok := body[from]
		if ok {
			into[to] = v
			delete(body, from)
		}
		return ok
	}

	contact, moved := section("contact"), false
	for _, key := range []string{"email", "phone", "url"} {
		moved = move(key, key, contact) || moved
	}
	if moved {
		body["contact"] = contact
	}

	billing, moved := section("carrier_billing"), false
	moved = move("carrier_billing_type", "type", billing) || moved
	moved = move("carrier_billing_account", "account_number", billing) || moved
	if moved {
		body["carrier_billing"] = billing
	}

	defaults, moved := section("defaults"), false
	for from, to := range preview9CustomerDefaults {
		moved = move(from, to, defaults) || moved
	}
	if days, ok := body["lead_time_days"]; ok {
		delete(body, "lead_time_days")
		defaults["lead_time"] = forgeLeadTime(days)
		moved = true
	}
	if policy, ok := body["fulfillment_policy"]; ok {
		delete(body, "fulfillment_policy")
		switch policy {
		case preview9MakeToStock:
			defaults["supply"] = string(constants.SupplyStock)
		case preview9MakeToOrder:
			defaults["supply"] = string(constants.SupplyMake)
		default:
			defaults["supply"] = policy
		}
		moved = true
	}
	if moved {
		body["defaults"] = defaults
	}
}

// forgeLeadTime is preview.9's whole days as forge's day Quantity; null stays null so the field still clears.
func forgeLeadTime(days any) any {
	switch d := days.(type) {
	case float64:
		return map[string]any{"value": strconv.FormatFloat(d, 'f', -1, 64), "unit_id": preview9DayUnitID}
	case string:
		return map[string]any{"value": d, "unit_id": preview9DayUnitID}
	}
	return days
}

// preview9CustomerFilters are preview.9's customer list filters that forge renamed.
var preview9CustomerFilters = map[string]string{
	"status_codes":            "standings",
	"customer_group_ids":      "group_ids",
	"commission_status_codes": "commission_policies",
	"freight_status_codes":    "freight_policies",
	"starts_at":               "created_after",
	"ends_at":                 "created_before",
}

// preview9CustomerIncludes are preview.9's include paths under a customer, by the forge path that replaced them.
// An empty forge path means the field is always present now, so the include is dropped.
var preview9CustomerIncludes = map[string]string{
	"bill_to_address":             "defaults.bill_to_address",
	"ship_to_address":             "defaults.ship_to_address",
	"type":                        "group",
	"parent_account":              "parent_customer",
	"child_accounts":              "child_customers",
	"freight_preferences.carrier": "defaults.carrier",
	"freight_preferences.carrier.service_levels": "defaults.carrier.service_levels",
	"freight_preferences.service_level":          "defaults.service_level",
	"notification_preferences":                   "notification_recipients",
	"freight_preferences":                        "",
	"defaults":                                   "",
	"defaults.priority":                          "",
	"contact_info":                               "",
}

func (customerForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route == customersRoute {
		for old, forge := range preview9CustomerFilters {
			for _, suffix := range []string{"", "[]"} {
				if values, ok := query[old+suffix]; ok {
					query.Del(old + suffix)
					query[forge+suffix] = append(query[forge+suffix], values...)
				}
			}
		}
		upgradeStandingFilter(query)
		upgradeParentFilter(query)
	}
	for _, param := range []string{"include", "include[]"} {
		values, ok := query[param]
		if !ok {
			continue
		}
		var kept []string
		for _, v := range values {
			for _, key := range strings.Split(v, ",") {
				if key = preview9CustomerIncludeKey(route, strings.TrimSpace(key)); key != "" {
					kept = append(kept, key)
				}
			}
		}
		switch {
		case len(kept) == 0:
			query.Del(param)
		case param == "include":
			query[param] = []string{strings.Join(kept, ",")}
		default:
			query[param] = kept
		}
	}
	return query
}

// preview9CustomerIncludeKey maps a preview.9 include key that reached into a customer to its forge key, when the
// route offers that. A key the route still offers, or that names nothing of a customer's, passes through.
func preview9CustomerIncludeKey(route, key string) string {
	if key == "" || routeOffers(route, key) {
		return key
	}
	segs := strings.Split(key, ".")
	for i := range segs {
		prefix, tail := strings.Join(segs[:i], "."), strings.Join(segs[i:], ".")
		forge, ok := preview9CustomerIncludes[tail]
		if !ok {
			continue
		}
		if forge == "" {
			return ""
		}
		if prefix != "" {
			forge = prefix + "." + forge
		}
		if routeOffers(route, forge) {
			return forge
		}
	}
	return key
}

// upgradeStandingFilter drops the retired preferred standing, which no customer holds any more.
func upgradeStandingFilter(query url.Values) {
	for _, key := range []string{"standings", "standings[]"} {
		values, ok := query[key]
		if !ok {
			continue
		}
		var kept []string
		for _, v := range values {
			if v != string(constants.AccountStatusCodePreferred) {
				kept = append(kept, v)
			}
		}
		if len(kept) == 0 {
			query.Del(key)
		} else {
			query[key] = kept
		}
	}
}

// upgradeParentFilter turns preview.9's parent_account_status into the relationship types it meant.
func upgradeParentFilter(query url.Values) {
	status := query.Get("parent_account_status")
	if status == "" {
		return
	}
	query.Del("parent_account_status")
	switch constants.CustomerParentAccountStatus(status) {
	case constants.CustomerParentAccountStatusParent:
		query.Add("relationship_types[]", string(constants.CustomerRelationshipTypeParent))
	case constants.CustomerParentAccountStatusNonParent:
		query.Add("relationship_types[]", string(constants.CustomerRelationshipTypeStandalone))
		query.Add("relationship_types[]", string(constants.CustomerRelationshipTypeChild))
	default:
		query.Add("relationship_types[]", status)
	}
}
