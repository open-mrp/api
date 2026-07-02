package resourceloaders

import (
	"testing"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	pb "github.com/augno/api/shared/proto/platform"
)

// The nested loader (used when a request_log is embedded as another resource's
// sub-resource, e.g. an audit event's "request") must scrub internal infra for
// agent requests just like the direct request-logs presenter does — otherwise an
// audit event's ?include=request would leak the internal host / pod IP.
func TestRequestLogFromProto_Loader_ScrubsInternalAgentInfra(t *testing.T) {
	const internalHost = "api-gateway-internal:8091"
	const podIP = "10.244.0.18"

	t.Run("agent request is scrubbed", func(t *testing.T) {
		got := requestLogFromProto(&pb.RequestLogInfo{
			Id:           "rq_agent",
			Method:       "GET",
			Host:         internalHost,
			Path:         "/v1/sales/customers",
			ClientIp:     new(podIP),
			IdentityType: new("agent"),
		})
		if got == nil {
			t.Fatal("expected a request log")
		}
		if got.Host != apiresource.RedactedRequestLogHost {
			t.Errorf("Host = %q, want %q", got.Host, apiresource.RedactedRequestLogHost)
		}
		if got.ClientIP != nil {
			t.Errorf("ClientIP = %v, want nil", *got.ClientIP)
		}
	})

	t.Run("user request keeps host and client_ip", func(t *testing.T) {
		got := requestLogFromProto(&pb.RequestLogInfo{
			Id:           "rq_user",
			Method:       "GET",
			Host:         "api.augno.com",
			Path:         "/v1/catalog/items",
			ClientIp:     new("198.51.100.7"),
			IdentityType: new("user"),
		})
		if got == nil {
			t.Fatal("expected a request log")
		}
		if got.Host != "api.augno.com" {
			t.Errorf("Host = %q, want api.augno.com", got.Host)
		}
		if got.ClientIP == nil || *got.ClientIP != "198.51.100.7" {
			t.Errorf("ClientIP = %v, want 198.51.100.7", got.ClientIP)
		}
	})
}
