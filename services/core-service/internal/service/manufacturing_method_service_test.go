package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
)

func TestOperationGroups(t *testing.T) {
	op := func(id string, seq constants.OperationSequence) *domain.MethodOperation {
		return &domain.MethodOperation{ID: id, Sequence: seq}
	}
	after, with := constants.OperationSequenceAfterPrevious, constants.OperationSequenceWithPrevious
	groups := operationGroups([]*domain.MethodOperation{op("a", after), op("b", with), op("c", after), op("d", after), op("e", with)})

	ids := make([][]string, len(groups))
	for i, g := range groups {
		for _, o := range g {
			ids[i] = append(ids[i], o.ID)
		}
	}
	assert.Equal(t, [][]string{{"a", "b"}, {"c"}, {"d", "e"}}, ids, "an operation alongside the previous one joins its group")

	// The first operation starts a group even when it says it runs with a previous one.
	assert.Len(t, operationGroups([]*domain.MethodOperation{op("x", with)}), 1)
}

func TestRequirePercent(t *testing.T) {
	assert.Nil(t, requirePercent("leveling_percent", "15"))
	assert.Nil(t, requirePercent("leveling_percent", "-50"))
	assert.NotNil(t, requirePercent("leveling_percent", "-100"))
	assert.NotNil(t, requirePercent("leveling_percent", "fifteen"))
}
