package main

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/openapi"
	"github.com/open-mrp/apikit/pagination"
)

// specConfig is the OpenAPI generator's configuration for OpenMRP: the spec header, the bearer-key scheme, how an
// endpoint's access policy reads in its description, and the sample data behind parameter and cursor examples.
func specConfig(groups []apiendpoint.APIEndpointGroup, publicOnly bool, transforms []openapi.Transform, version string) openapi.Config {
	return openapi.Config{
		Groups:      groups,
		Title:       "OpenMRP API",
		Description: "API for OpenMRP services",
		Version:     version,
		PublicOnly:  publicOnly,
		Servers: []openapi.Server{
			{URL: "https://api.openmrp.ai", Description: "Production server"},
			{URL: "http://localhost:8081", Description: "Local development server"},
		},
		SecuritySchemes: map[string]openapi.SecuritySchemeSpec{
			"BearerAuth": {
				Type:        "http",
				Scheme:      "bearer",
				Description: "API key authentication. Provide your API key as a Bearer token in the `Authorization` header.",
			},
		},
		// BearerAuth for API keys; the empty alternative allows cookie-based user sessions without Authorization.
		Security:     []map[string][]string{{"BearerAuth": {}}, {}},
		Transforms:   transforms,
		DescribeAuth: describeAuth,
		ParamDescriptions: map[string]string{
			"__Secure-openmrp.refresh-token": "The Secure refresh token cookie.",
		},
		Examples: openapi.Examples{
			PathParam:  pathParamExample,
			QueryParam: sampleQueryExampleForOpenAPIName,
			ListCursor: listCursorExample,
		},
		// LocationTypeCode is reused by Location.type and the location requests; as one named schema it does not collide with the LocationType resource in SDKs.
		NamedEnums: []reflect.Type{reflect.TypeFor[constants.LocationTypeCode]()},
	}
}

// describeAuth renders an endpoint's authz.Policy as the sentences that end its description: the role type and
// permissions it requires, after what acting in a customer's or supplier's account requires instead. The docs site
// reads the permissions from that final sentence.
func describeAuth(auth any) string {
	p, ok := authz.Of(auth)
	if !ok {
		return ""
	}
	requirement := ownAuthRequirement(p)
	if requirement == "" {
		return ""
	}
	if counterparty := counterpartyRequirement(p); counterparty != "" {
		return counterparty + "\n\n" + requirement
	}
	return requirement
}

func ownAuthRequirement(p authz.Policy) string {
	var sentences []string
	if p.RoleType != "" {
		sentences = append(sentences, fmt.Sprintf("This endpoint requires the `%s` role type.", p.RoleType))
	}
	if perms := policyPermissions(p); len(perms) > 0 {
		quoted := make([]string, len(perms))
		for i, perm := range perms {
			quoted[i] = "`" + perm + "`"
		}
		noun := "permission"
		if len(perms) > 1 {
			noun = "permissions"
		}
		list := strings.Join(quoted, ", ")
		if p.RequireAll && len(quoted) > 1 {
			list = strings.Join(quoted[:len(quoted)-1], ", ") + " and " + quoted[len(quoted)-1]
		}
		sentences = append(sentences, fmt.Sprintf("This endpoint requires the %s: %s.", noun, list))
	}
	return strings.Join(sentences, " ")
}

// counterpartyRequirement states what the policy's Counterparty asks for in a customer's or supplier's account, leaving out an account kind whose permission is the endpoint's own.
func counterpartyRequirement(p authz.Policy) string {
	own := policyPermissions(p)
	var clauses []string
	for _, kind := range []struct {
		perm    types.Permission
		account string
	}{{p.Counterparty.Customer, "a customer's"}, {p.Counterparty.Supplier, "a supplier's"}} {
		code := permissionString(kind.perm)
		if code == "" || slices.Contains(own, code) {
			continue
		}
		clauses = append(clauses, fmt.Sprintf("acting in %s account requires `%s`", kind.account, code))
	}
	if len(clauses) == 0 {
		return ""
	}
	sentence := strings.Join(clauses, ", and ")
	if len(clauses) > 1 {
		sentence += ","
	}
	sentence += " instead of the permission this endpoint requires in your own account."
	return strings.ToUpper(sentence[:1]) + sentence[1:]
}

// policyPermissions renders each of the policy's permissions as "<domain>:<action>".
func policyPermissions(p authz.Policy) []string {
	var out []string
	for _, perm := range p.Permissions {
		if s := permissionString(perm); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func permissionString(p types.Permission) string {
	dom, act := strings.TrimSpace(string(p.Domain)), strings.TrimSpace(string(p.Action))
	if dom == "" || act == "" {
		return ""
	}
	return dom + ":" + act
}

// pathParamExample picks a sample ID for a path parameter: by the request field it binds to, then by its name, then, for a generic {id}, by the resource the route names.
func pathParamExample(param, route, field string) string {
	if v := sampleIDFromFieldName(field); v != "" {
		return v
	}
	return sampleIDFromPathParam(param, route)
}

// listElementTypesUsingInt64Cursor are list item types whose backends paginate with
// pagination.Cursor (internal_id int64) rather than pagination.StringCursor.
var listElementTypesUsingInt64Cursor = map[string]struct{}{
	"APIKey":              {},
	"Sandbox":             {},
	"PricingPlan":         {},
	"RegistrationSession": {},
}

// listCursorExample signs the documented next-page cursor for the int64-keyed lists; the rest take the generator's string cursor.
func listCursorExample(itemType string, item map[string]any) string {
	if _, ok := listElementTypesUsingInt64Cursor[itemType]; !ok {
		return ""
	}
	occurredAt := apiresource.SampleAnalyticsPeriodStart
	for _, key := range []string{"created_at", "occurred_at"} {
		if s, ok := item[key].(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				occurredAt = t
				break
			}
		}
	}
	return pagination.EncodeDocumentationCursor(occurredAt, apiresource.SamplePaginationInternalID)
}

// endpointPolicy returns the authz.Policy an endpoint declares as its Auth.
func endpointPolicy(e apiendpoint.APIEndpointer) (authz.Policy, bool) {
	v := reflect.ValueOf(e)
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	f := v.FieldByName("Auth")
	if !f.IsValid() {
		return authz.Policy{}, false
	}
	return authz.Of(f.Interface())
}
