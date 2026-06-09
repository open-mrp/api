//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const receivablesPath = "/v1/finance/receivables"

func TestReceivables_List(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(receivablesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)
}

func TestReceivables_ListResponseShape(t *testing.T) {
	t.Parallel()
	list, _, err := apiClient.GetList(receivablesPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "list", list.Object)

	if len(list.Data) == 0 {
		t.Fatal("No receivable entries available to verify shape")
	}

	m := parseJSON(list.Data[0])
	assert.Equal(t, "receivable_entry", jsonField(m, "object"))

	invoice := jsonObject(m, "invoice")
	require.NotNil(t, invoice, "invoice should be a sub-resource")
	assert.Equal(t, "invoice", jsonField(invoice, "object"))
	assert.NotEmpty(t, jsonField(invoice, "id"))

	customer := jsonObject(m, "customer")
	require.NotNil(t, customer, "customer should be a sub-resource")
	assert.Equal(t, "customer", jsonField(customer, "object"))
	assert.NotEmpty(t, jsonField(customer, "id"))
}
