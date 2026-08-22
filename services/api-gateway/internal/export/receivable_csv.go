package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	pb "github.com/open-mrp/api/shared/proto/core"
)

const (
	CSVContentType = "text/csv"
)

// ReceivablesToCSV builds a CSV file from a list of receivable entry protos.
func ReceivablesToCSV(entries []*pb.ReceivableEntryProto) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	headers := []string{"Invoice Number", "Invoiced At", "Customer Number", "Customer Name", "Remaining Balance"}
	if err := w.Write(headers); err != nil {
		return nil, fmt.Errorf("csv header write: %w", err)
	}

	for _, e := range entries {
		invoicedAt := ""
		if e.InvoicedAt != nil {
			invoicedAt = e.InvoicedAt.AsTime().Format(time.RFC3339)
		}

		row := []string{
			e.InvoiceNumber,
			invoicedAt,
			e.CustomerNumber,
			e.CustomerName,
			e.RemainingBalance,
		}
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("csv row write: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv flush: %w", err)
	}

	return buf.Bytes(), nil
}
