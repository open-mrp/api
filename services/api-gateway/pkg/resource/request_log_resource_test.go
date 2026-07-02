package apiresource

import "testing"

func TestRequestLog_ScrubInternalInfra(t *testing.T) {
	const internalHost = "api-gateway-internal:8091"
	const podIP = "10.244.0.18"

	tests := []struct {
		name         string
		identityType *string
		wantScrub    bool
	}{
		{name: "agent request is scrubbed", identityType: new("agent"), wantScrub: true},
		{name: "user request is untouched", identityType: new("user"), wantScrub: false},
		{name: "api_key request is untouched", identityType: new("api_key"), wantScrub: false},
		{name: "nil identity type is untouched", identityType: nil, wantScrub: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rl := &RequestLog{Host: internalHost, ClientIP: new(podIP)}
			rl.ScrubInternalInfra(tc.identityType)

			if tc.wantScrub {
				if rl.Host != RedactedRequestLogHost {
					t.Errorf("Host = %q, want redacted %q", rl.Host, RedactedRequestLogHost)
				}
				if rl.Host == internalHost {
					t.Error("internal host leaked through scrub")
				}
				if rl.ClientIP != nil {
					t.Errorf("ClientIP = %v, want nil (pod IP must not leak)", *rl.ClientIP)
				}
			} else {
				if rl.Host != internalHost {
					t.Errorf("Host = %q, want preserved %q", rl.Host, internalHost)
				}
				if rl.ClientIP == nil || *rl.ClientIP != podIP {
					t.Errorf("ClientIP = %v, want preserved %q", rl.ClientIP, podIP)
				}
			}
		})
	}
}

// A nil receiver must be safe — presenters may call it on a freshly-built pointer.
func TestRequestLog_ScrubInternalInfra_NilReceiver(t *testing.T) {
	var rl *RequestLog
	rl.ScrubInternalInfra(new("agent")) // must not panic
}
