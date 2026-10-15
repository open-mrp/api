package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

func TestCustomerListPageQuery_EmitsOnlyTheFiltersSent(t *testing.T) {
	query, args := customerListPageQuery(domain.ListCustomersParams{AccountID: "ac_owner"}, customerListSearch{}, nil, nil, 26)
	assert.NotContains(t, query, "IS NULL OR")
	assert.NotContains(t, query, "= false OR")
	assert.NotContains(t, query, "JOIN account a", "no search, no join to the account")
	assert.Equal(t, []any{"ac_owner", int32(26)}, args)
}

func TestCustomerListPageQuery_Filters(t *testing.T) {
	active := false
	query, args := customerListPageQuery(domain.ListCustomersParams{
		AccountID:         "ac_owner",
		IsActive:          &active,
		StatusCodes:       []string{"hold_all"},
		RelationshipTypes: []constants.CustomerRelationshipType{constants.CustomerRelationshipTypeChild},
		City:              new("Raleigh"),
	}, customerListSearch{like: "%acme%", matches: []string{"ac_1"}}, nil, nil, 11)

	assert.Contains(t, query, "ar.counterparty_account_id IN (?)")
	assert.Contains(t, query, "COALESCE(NULLIF(ar.alias, ''), a.name) LIKE ? OR ar.external_number LIKE ?")
	assert.Contains(t, query, "ar.is_active = ?")
	assert.Contains(t, query, "ar.account_status_code IN (?)")
	assert.Contains(t, query, "ar.parent_account_relation_id IS NOT NULL AND NOT EXISTS")
	assert.Contains(t, query, "g.locality = ?")
	assert.NotContains(t, query, "g.state = ?", "only the address parts sent are matched")
	assert.NotContains(t, query, "notes", "a search never reads the notes")
	assert.True(t, strings.HasSuffix(query, "LIMIT ?"))
	assert.Equal(t, int32(11), args[len(args)-1])
}

func TestCustomerRelationshipClause(t *testing.T) {
	assert.Empty(t, customerRelationshipClause(nil))
	both := customerRelationshipClause([]constants.CustomerRelationshipType{
		constants.CustomerRelationshipTypeParent, constants.CustomerRelationshipTypeParent, constants.CustomerRelationshipTypeStandalone,
	})
	assert.Equal(t, 1, strings.Count(both, " OR "), "a repeated type is matched once")
}
