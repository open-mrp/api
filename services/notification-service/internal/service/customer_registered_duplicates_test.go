package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/messaging"
)

func TestPossibleDuplicatesSentence(t *testing.T) {
	northside := messaging.PossibleDuplicateCustomer{CustomerName: "Northside Pharmacy / Main", CustomerNumber: "104233", Reasons: []string{"same address", "same email domain"}}
	assert.Equal(t,
		"It may duplicate Northside Pharmacy / Main (#104233): same address and same email domain. If it does, merge it into that customer before its first order.",
		possibleDuplicatesSentence([]messaging.PossibleDuplicateCustomer{northside}))

	four := []messaging.PossibleDuplicateCustomer{{CustomerName: "A"}, {CustomerName: "B"}, {CustomerName: "C"}, {CustomerName: "D"}}
	assert.Equal(t,
		"It may duplicate A; B; C; and 1 more. If it does, merge it into that customer before its first order.",
		possibleDuplicatesSentence(four))
}
