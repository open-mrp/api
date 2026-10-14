package main

import (
	"testing"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
)

// TestCounterpartyPermissionsPairWithOneOwnPermission keeps an endpoint that names counterparty permissions to a single own permission, so every request it serves asks for exactly one.
func TestCounterpartyPermissionsPairWithOneOwnPermission(t *testing.T) {
	routed := 0
	for _, group := range openAPIEndpointGroups() {
		for _, e := range group.Endpoints {
			p, ok := endpointPolicy(e)
			if !ok || p.Counterparty == (authz.CounterpartyPermissions{}) {
				continue
			}
			routed++
			if perms := policyPermissions(p); len(perms) != 1 || p.RequireAll {
				t.Errorf("%s %s names counterparty permissions beside %v; it must declare exactly one own permission", e.GetMethod(), e.GetRoute(), perms)
			}
		}
	}
	if routed == 0 {
		t.Fatal("no endpoint names counterparty permissions")
	}
}
