package main

import "testing"

// TestCounterpartyPermissionsPairWithOneOwnPermission keeps an endpoint that names counterparty permissions to a single own permission, so every request it serves asks for exactly one.
func TestCounterpartyPermissionsPairWithOneOwnPermission(t *testing.T) {
	routed := 0
	for _, group := range openAPIEndpointGroups() {
		for _, e := range group.Endpoints {
			spec := endpointSpecField(e)
			if f := spec.FieldByName("CounterpartyPermissions"); !f.IsValid() || f.IsZero() {
				continue
			}
			routed++
			if perms := explicitPermissions(spec); len(perms) != 1 || requiresAllPermissions(spec) {
				t.Errorf("%s %s names counterparty permissions beside %v; it must declare exactly one own permission", e.GetMethod(), e.GetRoute(), perms)
			}
		}
	}
	if routed == 0 {
		t.Fatal("no endpoint names counterparty permissions")
	}
}
