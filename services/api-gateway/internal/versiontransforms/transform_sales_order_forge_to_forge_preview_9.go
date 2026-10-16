package versiontransforms

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped the sales order:
//   - status: estimate became draft, and canceled (stopped before anything shipped) is new; preview.9 reads canceled
//     as fulfilled. Issue, unissue and cancel are POSTs; preview.9 keeps its PUT issue | unissue | close | open routes.
//   - note became instructions, kept per party; priority and payment_intent_ids are gone (preview.9 reads null and []).
//   - bill_to and ship_to are copies of the addresses; preview.9 keeps bill_to_address and ship_to_address (the saved
//     addresses, internal on forge) and its contacts.
//   - freight is always present as { carrier, service_level, carrier_billing { type, account_number } }.
//   - commitment dates are business days (promised_on, ship_by_override_on, ship_by_on), lead_time_override is a day
//     Quantity, and the promised lead time is recorded.
//   - sales_rep is the account user, not an actor; related.production_run became related.jobs; completed_at became
//     fulfilled_at, first_ship_at first_shipped_at and expired_at the date expires_on.
//   - writes take customer_id, freight, commitment and email_recipients; list filters statuses[], group_ids and the
//     created range.
//
// preview.9 callers send and read the old shapes.
type salesOrderForgeToPreview9 struct {
	forgeToPreview9
	everyObject
}

func init() {
	version.RegisterTransformer(salesOrderForgeToPreview9{})
}

const (
	salesOrdersRoute = "/v1/sales/sales-orders"
	salesOrderRoute  = "/v1/sales/sales-orders/{id}"
)

// ForcedIncludes loads the rep's user, which the preview.9 actor's name and handle come from.
func (salesOrderForgeToPreview9) ForcedIncludes(objectType object.Type) []string {
	if objectType == constants.ObjectTypeSalesOrder {
		return []string{"sales_rep.user"}
	}
	return nil
}

func (salesOrderForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] == string(constants.ObjectTypeSalesOrder) {
			downgradeSalesOrder(obj)
		}
	})
	return data
}

func downgradeSalesOrder(obj map[string]any) {
	if _, forge := obj["currency"]; !forge {
		return
	}
	switch obj["status"] {
	case string(constants.SalesOrderStatusDraft):
		obj["status"] = string(constants.SalesOrderStatusCodeEstimate)
	case string(constants.SalesOrderStatusCanceled):
		obj["status"] = string(constants.SalesOrderStatusCodeFulfilled)
	}
	renameKey(obj, "instructions", "note")
	obj["priority"] = nil
	obj["payment_intent_ids"] = []any{}
	obj["sales_rep"] = preview9SalesRep(obj["sales_rep"])

	freight := map[string]any{"object": string(constants.ObjectTypeFreight), "policy": nil, "carrier": nil, "service_level": nil, "billing_type": nil, "billing_account_number": nil}
	if f, ok := obj["freight"].(map[string]any); ok {
		freight["carrier"], freight["service_level"] = f["carrier"], f["service_level"]
		if billing, ok := f["carrier_billing"].(map[string]any); ok {
			freight["billing_type"], freight["billing_account_number"] = billing["type"], billing["account_number"]
		}
	}
	obj["freight"] = freight

	obj["commitment"] = preview9Commitment(obj["commitment"], obj["issued_at"])
	renameKey(obj, "fulfilled_at", "completed_at")
	renameKey(obj, "first_shipped_at", "first_ship_at")
	obj["expired_at"] = preview9Instant(obj["expires_on"])
	delete(obj, "expires_on")

	if related, ok := obj["related"].(map[string]any); ok {
		var run any
		if jobs, ok := related["jobs"].(map[string]any); ok {
			if data, _ := jobs["data"].([]any); len(data) > 0 {
				run = data[0]
			}
		}
		related["production_run"] = run
		delete(related, "jobs")
	}
	for _, k := range []string{"currency", "site", "bill_to", "ship_to", "email_recipients"} {
		delete(obj, k)
	}
}

// preview9SalesRep is the account user credited with an order as preview.9's actor, named from the user loaded with
// it; nil stays nil.
func preview9SalesRep(v any) any {
	au, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	actor := map[string]any{"id": au["id"], "object": string(constants.ObjectTypeActor), "type": string(constants.ActorTypeUser),
		"name": nil, "handle": nil, "avatar_url": nil, "role": nil}
	if user, ok := au["user"].(map[string]any); ok {
		actor["name"], actor["handle"], actor["avatar_url"] = user["name"], user["email"], user["image_url"]
	}
	return actor
}

// preview9Commitment is a forge commitment in preview.9's shape: instants at midnight UTC for its dates, whole days
// for the override, and the ship-by date at the pickup cutoff when one was stamped.
func preview9Commitment(v, issuedAt any) any {
	c, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{
		"object":                   string(constants.ObjectTypeCommitment),
		"promised_at":              preview9Instant(c["promised_on"]),
		"lead_time_override_days":  preview9LeadTimeDays(c["lead_time_override"]),
		"ship_by_override_date":    preview9Instant(c["ship_by_override_on"]),
		"ship_by_date":             preview9Instant(c["ship_by_on"]),
		"lead_time_days":           nil,
		"lead_time_source":         c["lead_time_source"],
		"transit_days":             c["transit_days"],
		"transit_source":           c["transit_source"],
		"calendar_adjustment_days": c["calendar_adjustment_days"],
		"estimated_delivery_date":  nil,
	}
	if cutoff, ok := c["ship_by_cutoff_at"].(string); ok {
		out["ship_by_date"] = cutoff
	}
	shipBy, _ := c["ship_by_on"].(string)
	issued, _ := issuedAt.(string)
	if day, err := time.Parse(time.DateOnly, shipBy); err == nil {
		if at, err := time.Parse(time.RFC3339Nano, issued); err == nil {
			y, m, d := at.UTC().Date()
			out["lead_time_days"] = int(day.Sub(time.Date(y, m, d, 0, 0, 0, 0, time.UTC)).Hours() / 24)
		}
	}
	return out
}

// preview9Instant is a YYYY-MM-DD day as the midnight-UTC instant preview.9 read it as; anything else is nil.
func preview9Instant(v any) any {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	day, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil
	}
	return day.Format(time.RFC3339)
}

// forgeDay is a preview.9 instant as the UTC day forge takes; null stays null and anything unparseable passes
// through for validation to reject.
func forgeDay(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return v
	}
	return t.UTC().Format(time.DateOnly)
}

func (salesOrderForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

func (salesOrderForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	switch {
	case method == http.MethodPost && route == salesOrdersRoute:
		renameKey(data, "buyer_account_id", "customer_id")
		upgradeSalesOrderWrite(data, false)
	case method == http.MethodPatch && route == salesOrderRoute:
		upgradeSalesOrderWrite(data, true)
	}
	return data
}

// upgradeSalesOrderWrite moves a preview.9 order write's flat fields into forge's sections. On an update, the
// per-document contact lists stay as they are: they replace one kind of recipient alone.
func upgradeSalesOrderWrite(body map[string]any, update bool) {
	renameKey(body, "note", "instructions")
	delete(body, "priority_code")

	section := func(name string) map[string]any {
		m, _ := body[name].(map[string]any)
		if m == nil {
			m = map[string]any{}
		}
		return m
	}
	freight, moved := section("freight"), false
	for _, k := range []string{"carrier_id", "service_level_id"} {
		if v, ok := body[k]; ok {
			freight[k] = v
			delete(body, k)
			moved = true
		}
	}
	billingType, hasType := body["carrier_billing_type"]
	account, hasAccount := body["carrier_billing_account_number"]
	delete(body, "carrier_billing_type")
	delete(body, "carrier_billing_account_number")
	switch {
	case hasType && billingType == nil:
		freight["carrier_billing"] = nil
		moved = true
	case hasType:
		billing := map[string]any{"type": billingType}
		if hasAccount && account != nil {
			billing["account_number"] = account
		}
		freight["carrier_billing"] = billing
		moved = true
	}
	if moved {
		body["freight"] = freight
	}

	commitment, moved := section("commitment"), false
	for from, to := range map[string]string{"promised_at": "promised_on", "ship_by_override_date": "ship_by_override_on"} {
		if v, ok := body[from]; ok {
			commitment[to] = forgeDay(v)
			delete(body, from)
			moved = true
		}
	}
	if v, ok := body["lead_time_override_days"]; ok {
		commitment["lead_time_override"] = forgeLeadTime(v)
		delete(body, "lead_time_override_days")
		moved = true
	}
	if moved {
		body["commitment"] = commitment
	}

	for _, side := range []struct{ id, inline, to string }{
		{"bill_to_address_id", "bill_to_address", "bill_to"},
		{"ship_to_address_id", "ship_to_address", "ship_to"},
		{"billing_address_id", "billing_address", "bill_to"},
		{"shipping_address_id", "shipping_address", "ship_to"},
	} {
		if id, ok := body[side.id]; ok {
			body[side.to] = map[string]any{"address_id": id}
			delete(body, side.id)
		}
		if inline, ok := body[side.inline].(map[string]any); ok {
			body[side.to] = forgeInlineAddress(inline)
			delete(body, side.inline)
		}
	}

	if update {
		return
	}
	ack, hasAck := body["acknowledgement_email_contacts"].([]any)
	inv, hasInv := body["invoice_email_contacts"].([]any)
	delete(body, "acknowledgement_email_contacts")
	delete(body, "invoice_email_contacts")
	if hasAck || hasInv {
		body["email_recipients"] = mergeOrderContacts(ack, inv)
	}
}

// forgeInlineAddress is a preview.9 inline address as forge's copied fields. A saved address it named is no longer
// edited by the order; the order keeps its own copy.
func forgeInlineAddress(in map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"name", "line_1", "line_2", "city", "state", "postal_code", "country", "phone", "email"} {
		if v, ok := in[k]; ok && v != nil {
			out[k] = v
		}
	}
	// The address transformer may not have renamed its fields yet.
	for from, to := range map[string]string{"street_line_1": "line_1", "street_line_2": "line_2", "locality": "city"} {
		if v, ok := in[from]; ok && v != nil {
			out[to] = v
		}
	}
	return out
}

// mergeOrderContacts turns preview.9's per-document contact lists into forge's recipients, one per account user.
func mergeOrderContacts(ack, inv []any) []any {
	var order []string
	docs := map[string][]any{}
	add := func(list []any, doc constants.RecipientDocument) {
		for _, c := range list {
			m, _ := c.(map[string]any)
			id, _ := m["account_user_id"].(string)
			if id == "" {
				continue
			}
			if _, seen := docs[id]; !seen {
				order = append(order, id)
			}
			docs[id] = append(docs[id], string(doc))
		}
	}
	add(ack, constants.RecipientDocumentAcknowledgment)
	add(inv, constants.RecipientDocumentInvoice)
	out := make([]any, 0, len(order))
	for _, id := range order {
		out = append(out, map[string]any{"account_user_id": id, "documents": docs[id]})
	}
	return out
}

// preview9SalesOrderFilters are preview.9's sales order list filters that forge renamed.
var preview9SalesOrderFilters = map[string]string{
	"customer_group_ids": "group_ids",
	"starts_at":          "created_after",
	"ends_at":            "created_before",
}

// preview9SalesOrderIncludes are preview.9's include paths under an order, by the forge paths that replaced them.
var preview9SalesOrderIncludes = map[string][]string{
	"freight":                {"freight.carrier", "freight.service_level"},
	"related.production_run": {"related.jobs"},
}

func (salesOrderForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	if route == salesOrdersRoute {
		for old, forge := range preview9SalesOrderFilters {
			for _, suffix := range []string{"", "[]"} {
				if values, ok := query[old+suffix]; ok {
					query.Del(old + suffix)
					query[forge+suffix] = append(query[forge+suffix], values...)
				}
			}
		}
		for _, suffix := range []string{"", "[]"} {
			values, ok := query["status_codes"+suffix]
			if !ok {
				continue
			}
			query.Del("status_codes" + suffix)
			for _, v := range values {
				if v != "all" {
					query.Add("statuses"+suffix, string(constants.SalesOrderStatusCode(v).Status()))
				}
			}
		}
	}
	for _, param := range []string{"include", "include[]"} {
		values, ok := query[param]
		if !ok {
			continue
		}
		var kept []string
		for _, v := range values {
			for _, key := range strings.Split(v, ",") {
				kept = append(kept, preview9SalesOrderIncludeKeys(route, strings.TrimSpace(key))...)
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

// preview9SalesOrderIncludeKeys maps a preview.9 include key reaching into an order to the forge keys the route
// offers. A key the route still offers, or that names nothing of an order's, passes through.
func preview9SalesOrderIncludeKeys(route, key string) []string {
	if key == "" || routeOffers(route, key) {
		return []string{key}
	}
	segs := strings.Split(key, ".")
	for i := range segs {
		prefix, tail := strings.Join(segs[:i], "."), strings.Join(segs[i:], ".")
		forge, ok := preview9SalesOrderIncludes[tail]
		if !ok {
			continue
		}
		var out []string
		for _, f := range forge {
			if prefix != "" {
				f = prefix + "." + f
			}
			if routeOffers(route, f) {
				out = append(out, f)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{key}
}
