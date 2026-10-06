package service

import "testing"

// A From on the send payload is honored only for platform domains; anything else must fall back to noreply@ so a payload can never send as a tenant's domain.
func TestIsPlatformSender(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"dane@openmrp.ai":                  true,
		"Dane Albaugh <dane@openmrp.ai>":   true,
		"support@AUGNO.com":                true,
		"sales@tenant-example.com":         false,
		"dane@openmrp.ai.evil-example.com": false,
		"not an address":                   false,
		"":                                 false,
	}
	for from, want := range cases {
		from := from
		if got := isPlatformSender(&from); got != want {
			t.Errorf("isPlatformSender(%q) = %v, want %v", from, got, want)
		}
	}
	if isPlatformSender(nil) {
		t.Error("isPlatformSender(nil) = true, want false")
	}
}
