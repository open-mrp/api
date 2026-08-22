package service

import (
	"bytes"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOrderAcknowledgementPDF(t *testing.T) {
	t.Parallel()

	billName := "Acme Bill-To"
	billLine1 := "1 Main St"
	billCity := "Springfield"
	billState := "IL"
	billZip := "62701"
	carrier := "UPS"
	paymentTerm := "Net 30"
	desc := "Widget, 6061-T6"

	order := &domain.SalesOrder{
		Number:            "123",
		CustomerName:      "Acme Corp",
		CustomerNumber:    "C-1",
		PriorityName:      "Normal",
		CarrierName:       &carrier,
		PaymentTermName:   &paymentTerm,
		BillToName:        &billName,
		BillToStreetLine1: &billLine1,
		BillToLocality:    &billCity,
		BillToState:       &billState,
		BillToPostalCode:  &billZip,
		CreatedAt:         time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
	}
	lines := []*domain.SalesOrderLine{
		{ProductSKU: "SKU-1", ProductDescription: &desc, QuantityValue: "3", QuantityUnitAbbreviation: "ea", UnitPriceValue: "20.00"},
		{ProductSKU: "SKU-2", QuantityValue: "1.5", QuantityUnitAbbreviation: "lb", UnitPriceValue: "10.50"},
	}

	account := &domain.Account{Name: "Seller Co"}
	data := buildOrderAcknowledgementData(order, lines, account, nil)
	pdfBytes, err := buildOrderAcknowledgementPDF(data)
	require.NoError(t, err)
	require.NotEmpty(t, pdfBytes)
	// A well-formed PDF starts with the "%PDF" magic header.
	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF")), "output should be a valid PDF")
}

func TestBuildOrderAcknowledgementPDF_NoLinesNoAddresses(t *testing.T) {
	t.Parallel()
	// Must not panic on a bare order with no lines / no addresses.
	order := &domain.SalesOrder{Number: "1", CreatedAt: time.Now().UTC()}
	data := buildOrderAcknowledgementData(order, nil, nil, nil)
	pdfBytes, err := buildOrderAcknowledgementPDF(data)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF")))
}
