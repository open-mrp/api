package versiontransforms

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/open-mrp/api/shared/apiversion"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge reshaped the email bridge:
//   - email_domain: dkim_tokens and the mail_from_* strings became dns_records [{type, name, value}].
//   - email_inbox: group_id became the expandable group.
//   - email_sender: email_domain_id, domain and domain_status became the expandable email_domain; PUT became PATCH,
//     and DELETE answers with the deleted stub.
//   - the domain and inbox lists page (25 by default) where preview.9 returned every row.
//
// preview.9 callers get the old fields back from the forced includes and the records, and their PUT is served by
// the PATCH with every omitted optional field cleared, as a replacement did.
type emailBridgeForgeToPreview9 struct {
	forgeToPreview9
}

func init() {
	version.RegisterTransformer(emailBridgeForgeToPreview9{})
}

func (emailBridgeForgeToPreview9) ObjectTypes() []object.Type {
	return []object.Type{constants.ObjectTypeEmailDomain, constants.ObjectTypeEmailInbox, constants.ObjectTypeEmailSender}
}

func (emailBridgeForgeToPreview9) ForcedIncludes(objectType object.Type) []string {
	switch objectType {
	case constants.ObjectTypeEmailInbox:
		return []string{"group"}
	case constants.ObjectTypeEmailSender:
		return []string{"email_domain"}
	}
	return nil
}

func (emailBridgeForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		switch obj["object"] {
		case string(constants.ObjectTypeEmailDomain):
			downgradeEmailDomainRecords(obj)
		case string(constants.ObjectTypeEmailInbox):
			if _, ok := obj["group"]; ok {
				obj["group_id"] = idOf(obj["group"])
				delete(obj, "group")
			}
		case string(constants.ObjectTypeEmailSender):
			if dom, ok := obj["email_domain"].(map[string]any); ok {
				obj["email_domain_id"] = dom["id"]
				obj["domain"] = dom["domain"]
				obj["domain_status"] = dom["status"]
			}
			delete(obj, "email_domain")
		}
	})
	return data
}

// downgradeEmailDomainRecords rebuilds dkim_tokens and the mail_from_* fields from dns_records. A DKIM CNAME is named <token>._domainkey.<domain>; the MX and TXT records sit on the MAIL FROM subdomain.
func downgradeEmailDomainRecords(obj map[string]any) {
	records, ok := obj["dns_records"].([]any)
	if !ok {
		return
	}
	tokens := []any{}
	var mailFrom, mx, spf any
	for _, r := range records {
		rec, ok := r.(map[string]any)
		if !ok {
			continue
		}
		name, _ := rec["name"].(string)
		switch rec["type"] {
		case string(constants.EmailDNSRecordTypeCNAME):
			if token, _, found := strings.Cut(name, "._domainkey."); found {
				tokens = append(tokens, token)
			}
		case string(constants.EmailDNSRecordTypeMX):
			mailFrom, mx = name, rec["value"]
		case string(constants.EmailDNSRecordTypeTXT):
			mailFrom, spf = name, rec["value"]
		}
	}
	obj["dkim_tokens"] = tokens
	obj["mail_from_domain"] = mailFrom
	obj["mail_from_mx_record"] = mx
	obj["mail_from_spf_record"] = spf
	delete(obj, "dns_records")
}

// TransformRequestBody upgrades preview.9's PUT /v1/messaging/email-sender, now served by PATCH: a field the
// replacement left out was cleared, so it is sent as null.
func (emailBridgeForgeToPreview9) TransformRequestBody(objectType object.Type, method, route string, data map[string]any) map[string]any {
	if objectType != constants.ObjectTypeEmailSender || method != http.MethodPatch || route != emailSenderRoute {
		return data
	}
	for _, key := range []string{"from_name", "reply_to"} {
		if _, ok := data[key]; !ok {
			data[key] = nil
		}
	}
	return data
}

func (emailBridgeForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

// TransformQuery keeps preview.9's whole-list reads of domains and inboxes whole for any account with up to 100 of them.
func (emailBridgeForgeToPreview9) TransformQuery(_ object.Type, route string, query url.Values) url.Values {
	return preview9UnpagedLimit(route, query)
}

// preview9UnpagedRoutes are the lists that returned every row on preview.9.
var preview9UnpagedRoutes = map[string]bool{
	"/v1/messaging/email-domains":            true,
	"/v1/messaging/email-inboxes":            true,
	"/v1/messaging/groups":                   true,
	"/v1/messaging/blocks":                   true,
	"/v1/messaging/conversations/{id}/links": true,
}

func preview9UnpagedLimit(route string, query url.Values) url.Values {
	if preview9UnpagedRoutes[route] && query.Get("limit") == "" {
		query.Set("limit", "100")
	}
	return query
}

// idOf is the id of an expanded reference, or nil.
func idOf(v any) any {
	if m, ok := v.(map[string]any); ok {
		return m["id"]
	}
	return nil
}

const emailSenderRoute = "/v1/messaging/email-sender"

// RoutePreview9Messaging serves preview.9's PUT /v1/messaging/email-sender from the forge PATCH. Only a request
// naming preview.9 is rerouted. It wraps the router, since a route is matched before any endpoint's transformers run.
func RoutePreview9Messaging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == emailSenderRoute && r.Header.Get(apiversion.Header) == apiversion.V1_0_Forge_Preview9.String() {
			r = r.Clone(r.Context())
			r.Method = http.MethodPatch
		}
		next.ServeHTTP(w, r)
	})
}
