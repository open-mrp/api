package domain

import (
	"strings"
	"testing"
)

// The display name is account-supplied and lands in a header of every merchant-facing message, so a name carrying CR/LF must not be able to terminate the From header and inject one of its own.
func TestAccountEmailSenderFromHeader(t *testing.T) {
	sender := func(name string) *AccountEmailSender {
		s := &AccountEmailSender{LocalPart: "orders", Domain: "carolon.com"}
		if name != "" {
			s.FromName = &name
		}
		return s
	}

	tests := []struct {
		name     string
		fromName string
		want     string
	}{
		{"no display name sends the bare address", "", "orders@carolon.com"},
		{"plain name is quoted", "Carolon Co.", `"Carolon Co." <orders@carolon.com>`},
		{"embedded quotes are escaped", `He said "hi"`, `"He said \"hi\"" <orders@carolon.com>`},
		{"angle brackets cannot open a second address", "a<b>c", `"a<b>c" <orders@carolon.com>`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sender(test.fromName).FromHeader(); got != test.want {
				t.Errorf("FromHeader() = %q, want %q", got, test.want)
			}
		})
	}

	t.Run("CRLF is encoded, not passed through", func(t *testing.T) {
		got := sender("evil\r\nBcc: attacker@evil.com").FromHeader()
		for _, forbidden := range []string{"\r", "\n"} {
			if strings.Contains(got, forbidden) {
				t.Fatalf("FromHeader() = %q, must not carry a raw line break", got)
			}
		}
		if !strings.Contains(got, "<orders@carolon.com>") {
			t.Errorf("FromHeader() = %q, want the real address preserved", got)
		}
	})
}
