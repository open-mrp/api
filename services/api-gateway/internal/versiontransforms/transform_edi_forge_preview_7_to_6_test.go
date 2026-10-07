package versiontransforms

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/version"
)

func upgradePreview6Body(objectType constants.ObjectType, method, route string, data map[string]any) map[string]any {
	return version.TransformEndpointRequest(version.V1_0_Forge_Preview6, version.V1_0_Forge_Preview7, objectType, method, route, data)
}

func TestEdiPreview7To6_DropsEdiStatusFromCustomerWrites(t *testing.T) {
	for _, tc := range []struct{ method, route string }{
		{http.MethodPost, "/v1/sales/customers"},
		{http.MethodPatch, "/v1/sales/customers/{id}"},
	} {
		got := upgradePreview6Body(constants.ObjectTypeCustomer, tc.method, tc.route,
			map[string]any{"name": "Acme", "edi_status": "enabled"})
		if want := map[string]any{"name": "Acme"}; !reflect.DeepEqual(got, want) {
			t.Errorf("%s %s: got %v, want %v", tc.method, tc.route, got, want)
		}
	}
}

func TestEdiPreview7To6_DropsIsEdiSentFromInvoiceUpdates(t *testing.T) {
	got := upgradePreview6Body(constants.ObjectTypeInvoice, http.MethodPatch, "/v1/finance/invoices/{id}",
		map[string]any{"note": "x", "is_edi_sent": true})
	if want := map[string]any{"note": "x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestEdiPreview7To6_LeavesOtherEndpointsAndFieldsAlone(t *testing.T) {
	body := map[string]any{"edi_status": "enabled", "name": "Acme"}
	got := upgradePreview6Body(constants.ObjectTypeCustomer, http.MethodPost, "/v1/sales/customers/actions/import", body)
	if want := map[string]any{"edi_status": "enabled", "name": "Acme"}; !reflect.DeepEqual(got, want) {
		t.Errorf("an endpoint that never took the field is not rewritten: got %v", got)
	}
}

func TestEdiPreview7To6_ResponsesPassThrough(t *testing.T) {
	customer := map[string]any{"id": "ac_1", "object": "customer", "name": "Acme"}
	got := version.Transform(version.V1_0_Forge_Preview7, version.V1_0_Forge_Preview6, constants.ObjectTypeCustomer, customer)
	if want := map[string]any{"id": "ac_1", "object": "customer", "name": "Acme"}; !reflect.DeepEqual(got, want) {
		t.Errorf("no value is invented for the dropped field: got %v", got)
	}
}
