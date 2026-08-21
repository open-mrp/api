// Package infra holds tests that assert security-critical properties of the
// Kubernetes manifests — chiefly that the api-gateway internal listener (port
// 8091, used by agent-service for endpoint-tools) is reachable ONLY from inside
// the cluster via agent-service, and never exposed to the public internet.
package infra

import (
	"os"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const internalPort = 8091

// environments under test. Production manifests live in the private augno/infra repo and are
// covered by the sibling copy of this test there; both sets must satisfy the same invariants.
var environments = []string{"development"}

type doc = map[string]any

func decodeAll(t *testing.T, path string) []doc {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	var docs []doc
	dec := yaml.NewDecoder(f)
	for {
		var d doc
		err := dec.Decode(&d)
		if err != nil {
			break
		}
		if len(d) > 0 {
			docs = append(docs, d)
		}
	}
	if len(docs) == 0 {
		t.Fatalf("no YAML documents decoded from %s", path)
	}
	return docs
}

func kindName(d doc) (string, string) {
	kind, _ := d["kind"].(string)
	name := ""
	if md, ok := d["metadata"].(doc); ok {
		name, _ = md["name"].(string)
	}
	return kind, name
}

func find(docs []doc, kind, name string) doc {
	for _, d := range docs {
		k, n := kindName(d)
		if k == kind && n == name {
			return d
		}
	}
	return nil
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

func asDoc(v any) doc {
	d, _ := v.(doc)
	return d
}

// servicePorts returns the integer ports exposed by a Service doc.
func servicePorts(svc doc) []int {
	var out []int
	spec := asDoc(svc["spec"])
	for _, p := range asList(spec["ports"]) {
		if pd := asDoc(p); pd != nil {
			if port, ok := pd["port"].(int); ok {
				out = append(out, port)
			}
		}
	}
	return out
}

// TestInternalServiceIsClusterIPOnly asserts the internal Service exists, is
// ClusterIP, and exposes only the internal port.
func TestInternalServiceIsClusterIPOnly(t *testing.T) {
	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			docs := decodeAll(t, env+"/kubernetes/apps/api-gateway.yaml")
			svc := find(docs, "Service", "api-gateway-internal")
			if svc == nil {
				t.Fatal("api-gateway-internal Service not found")
			}
			spec := asDoc(svc["spec"])
			if typ, _ := spec["type"].(string); typ != "ClusterIP" {
				t.Errorf("api-gateway-internal type = %q, want ClusterIP (must not be LoadBalancer/NodePort)", typ)
			}
			ports := servicePorts(svc)
			if len(ports) != 1 || ports[0] != internalPort {
				t.Errorf("api-gateway-internal ports = %v, want only [%d]", ports, internalPort)
			}
		})
	}
}

// TestPublicServiceDoesNotExposeInternalPort asserts the externally-typed
// api-gateway Service (LoadBalancer in dev, NodePort in prod) never exposes 8091.
func TestPublicServiceDoesNotExposeInternalPort(t *testing.T) {
	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			docs := decodeAll(t, env+"/kubernetes/apps/api-gateway.yaml")
			svc := find(docs, "Service", "api-gateway")
			if svc == nil {
				t.Fatal("api-gateway Service not found")
			}
			spec := asDoc(svc["spec"])
			typ, _ := spec["type"].(string)
			if typ != "LoadBalancer" && typ != "NodePort" {
				t.Fatalf("expected public api-gateway Service to be LoadBalancer/NodePort, got %q", typ)
			}
			if ports := servicePorts(svc); slices.Contains(ports, internalPort) {
				t.Errorf("public %s Service exposes internal port %d (ports=%v) — must not", typ, internalPort, ports)
			}
		})
	}
}

// TestNoIngressExposesInternalListener asserts no Ingress routes to the internal
// Service or its port, and that no ALB listen-ports annotation mentions 8091.
func TestNoIngressExposesInternalListener(t *testing.T) {
	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			docs := decodeAll(t, env+"/kubernetes/apps/api-gateway.yaml")
			for _, d := range docs {
				kind, name := kindName(d)
				if kind != "Ingress" {
					continue
				}

				// No annotation may reference the internal port (e.g. ALB listen-ports).
				if md := asDoc(d["metadata"]); md != nil {
					for k, v := range asDoc(md["annotations"]) {
						if s, ok := v.(string); ok && strings.Contains(s, "8091") {
							t.Errorf("Ingress %s annotation %q references internal port: %s", name, k, s)
						}
					}
				}

				// No backend may route to the internal Service.
				spec := asDoc(d["spec"])
				for _, rule := range asList(spec["rules"]) {
					http := asDoc(asDoc(rule)["http"])
					for _, p := range asList(http["paths"]) {
						backend := asDoc(asDoc(p)["backend"])
						bsvc := asDoc(backend["service"])
						if svcName, _ := bsvc["name"].(string); svcName == "api-gateway-internal" {
							t.Errorf("Ingress %s routes to api-gateway-internal — internal listener must not be public", name)
						}
						port := asDoc(bsvc["port"])
						if pn, ok := port["number"].(int); ok && pn == internalPort {
							t.Errorf("Ingress %s backend targets port %d — internal listener must not be public", name, internalPort)
						}
					}
				}
			}
		})
	}
}

// ruleMatchesAppOnPort reports whether a NetworkPolicy rule (ingress or egress)
// allows the given port and, via peers under peerKey ("from"/"to"), references
// ONLY a podSelector for the given app label.
func ruleMatchesAppOnPort(rule doc, peerKey, app string, port int) bool {
	hasPort := false
	for _, p := range asList(rule["ports"]) {
		if pd := asDoc(p); pd != nil {
			if pn, ok := pd["port"].(int); ok && pn == port {
				hasPort = true
			}
		}
	}
	if !hasPort {
		return false
	}
	peers := asList(rule[peerKey])
	if len(peers) == 0 {
		return false // a rule with no peers allows all sources — not app-scoped
	}
	for _, peer := range peers {
		sel := asDoc(asDoc(peer)["podSelector"])
		labels := asDoc(sel["matchLabels"])
		if a, _ := labels["app"].(string); a != app {
			return false
		}
	}
	return true
}

// portInAnyRule reports whether the given port appears in any rule under listKey.
func portHasOpenRule(np doc, listKey string, port int) bool {
	spec := asDoc(np["spec"])
	for _, r := range asList(spec[listKey]) {
		rule := asDoc(r)
		hasPort := false
		for _, p := range asList(rule["ports"]) {
			if pd := asDoc(p); pd != nil {
				if pn, ok := pd["port"].(int); ok && pn == port {
					hasPort = true
				}
			}
		}
		peerKey := "from"
		if listKey == "egress" {
			peerKey = "to"
		}
		if hasPort && len(asList(rule[peerKey])) == 0 {
			return true // port allowed from/to all peers
		}
	}
	return false
}

// TestNetworkPolicyLocksInternalPortToAgentService asserts the api-gateway
// NetworkPolicy allows 8091 ingress ONLY from agent-service, and never via an
// allow-all rule.
func TestNetworkPolicyLocksInternalPortToAgentService(t *testing.T) {
	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			docs := decodeAll(t, env+"/kubernetes/network-policies/api-gateway.yaml")
			np := find(docs, "NetworkPolicy", "api-gateway-ingress")
			if np == nil {
				t.Fatal("api-gateway-ingress NetworkPolicy not found")
			}
			spec := asDoc(np["spec"])

			if portHasOpenRule(np, "ingress", internalPort) {
				t.Errorf("internal port %d is allowed from all sources — must be agent-service only", internalPort)
			}

			locked := false
			for _, r := range asList(spec["ingress"]) {
				if ruleMatchesAppOnPort(asDoc(r), "from", "agent-service", internalPort) {
					locked = true
				}
			}
			if !locked {
				t.Errorf("no ingress rule restricts port %d to podSelector app=agent-service", internalPort)
			}
		})
	}
}

// TestAgentServiceMayReachInternalPort asserts agent-service egress explicitly
// allows the api-gateway internal port (so the policy does not block the
// legitimate path once enforcement is on).
func TestAgentServiceMayReachInternalPort(t *testing.T) {
	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			docs := decodeAll(t, env+"/kubernetes/network-policies/agent-service.yaml")
			np := find(docs, "NetworkPolicy", "agent-service-egress")
			if np == nil {
				t.Fatal("agent-service-egress NetworkPolicy not found")
			}
			spec := asDoc(np["spec"])
			allowed := false
			for _, r := range asList(spec["egress"]) {
				if ruleMatchesAppOnPort(asDoc(r), "to", "api-gateway", internalPort) {
					allowed = true
				}
			}
			if !allowed {
				t.Errorf("agent-service egress does not allow api-gateway:%d — endpoint-tools would be blocked under enforcement", internalPort)
			}
		})
	}
}

// TestConfigPointsAtInternalService asserts app-config wires the agent at the
// internal ClusterIP Service on the internal port.
func TestConfigPointsAtInternalService(t *testing.T) {
	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			docs := decodeAll(t, env+"/kubernetes/config/app-config.yaml")
			cm := find(docs, "ConfigMap", "app-config")
			if cm == nil {
				t.Fatal("app-config ConfigMap not found")
			}
			data := asDoc(cm["data"])
			url, _ := data["API_GATEWAY_INTERNAL_URL"].(string)
			if !strings.Contains(url, "api-gateway-internal:8091") {
				t.Errorf("API_GATEWAY_INTERNAL_URL = %q, want it to target api-gateway-internal:8091", url)
			}
			if p, _ := data["INTERNAL_PORT"].(string); p != "8091" {
				t.Errorf("INTERNAL_PORT = %q, want 8091", p)
			}
		})
	}
}
