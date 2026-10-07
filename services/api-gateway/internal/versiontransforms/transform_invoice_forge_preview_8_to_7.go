package versiontransforms

import (
	"net/url"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func init() {
	version.Register(&invoiceForgePreview8To7{})
}

// invoiceForgePreview8To7 bridges the invoice list's search from 1.0.forge-preview.8 to 1.0.forge-preview.7.
//
// preview.8 matches `q` against the start of each number and name it searches and leaves notes out, unless
// `q_match=contains` is sent. preview.7 matched anywhere in every field, notes included, so a preview.7 search
// asks for that explicitly. The resource itself did not change.
type invoiceForgePreview8To7 struct{}

// invoiceListRoute is the only invoice endpoint that searches; the others reject parameters they do not know.
const invoiceListRoute = "/v1/finance/invoices"

func (t *invoiceForgePreview8To7) FromVersion() version.APIVersion {
	return version.V1_0_Forge_Preview8
}

func (t *invoiceForgePreview8To7) ToVersion() version.APIVersion {
	return version.V1_0_Forge_Preview7
}

func (t *invoiceForgePreview8To7) ObjectTypes() []constants.ObjectType {
	return []constants.ObjectType{constants.ObjectTypeInvoice}
}

func (t *invoiceForgePreview8To7) Transform(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *invoiceForgePreview8To7) TransformRequest(_ constants.ObjectType, data map[string]any) map[string]any {
	return data
}

func (t *invoiceForgePreview8To7) TransformQuery(_ constants.ObjectType, route string, query url.Values) url.Values {
	if route != invoiceListRoute || query.Get("q") == "" || query.Has("q_match") {
		return query
	}
	query.Set("q_match", string(constants.InvoiceSearchMatchContains))
	return query
}
