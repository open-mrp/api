package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
	"github.com/open-mrp/api/shared/textutil"
)

// customerExportPageSize is one list page at the list's own ceiling.
const customerExportPageSize = 1000

const customerExportDateFormat = "mm/dd/yyyy"

// customerExportRow is a listed customer plus what its list row does not carry.
type customerExportRow struct {
	customer   *domain.Customer
	statusName string
	contacts   []domain.CustomerContact
}

// customerExportColumns are the dashboard's former customer export, column for column.
var customerExportColumns = []excel.ColumnSpec{
	{Header: "Customer Number", Key: "customerNumber", Width: 18},
	{Header: "Name", Key: "name", Width: 25},
	{Header: "Email", Key: "email", Width: 25},
	{Header: "Phone", Key: "phone", Width: 15},
	{Header: "Status", Key: "status", Width: 12},
	{Header: "Customer Group", Key: "customerGroup", Width: 20},
	{Header: "Sales Rep", Key: "salesRep", Width: 20},
	{Header: "Priority", Key: "priority", Width: 12},
	{Header: "Payment Term", Key: "paymentTerm", Width: 18},
	{Header: "Shipping Term", Key: "shippingTerm", Width: 18},
	{Header: "Carrier", Key: "carrier", Width: 18},
	{Header: "Default Billing", Key: "defaultBilling", Width: 35},
	{Header: "Default Shipping", Key: "defaultShipping", Width: 35},
	{Header: "Contacts", Key: "contacts", Width: 30},
	{Header: "Commission Exempt", Key: "commissionExempt", Width: 18},
	{Header: "Freight Exempt", Key: "freightExempt", Width: 15},
	{Header: "Parent Account", Key: "parentAccount", Width: 15},
	{Header: "Note", Key: "note", Width: 30},
	{Header: "Created At", Key: "createdAt", Width: 15, NumFmt: customerExportDateFormat},
}

func (s *customerSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

func (s *customerSvcImpl) exportSpec() exportSpec[customerExportRow, domain.ListCustomersParams] {
	return exportSpec[customerExportRow, domain.ListCustomersParams]{
		PermissionDomain: types.PermissionDomainCustomers,
		Name:             "Customers",
		Slug:             "customers",
		ResourceType:     constants.ObjectTypeCustomer,
		Columns:          customerExportColumns,
		Fetch:            fetchCustomerExportRows,
		Project:          projectCustomerExportRow,
	}
}

// ExportCustomers stores only the list's filters; the paging and includes are the export's own.
func (s *customerSvcImpl) ExportCustomers(ctx context.Context, filters domain.ListCustomersParams) (*domain.Job, *apierror.APIError) {
	filters.AccountID, filters.Cursor, filters.Limit, filters.Includes = "", nil, 0, nil
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), filters)
}

func (s *customerSvcImpl) BuildExportCustomers(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

// fetchCustomerExportRows walks the customer list itself, so an export holds exactly the customers the list shows for the same filters, in the same order.
func fetchCustomerExportRows(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ListCustomersParams) ([]customerExportRow, *apierror.APIError) {
	statusNames, apiErr := accountStatusNames(ctx, repos)
	if apiErr != nil {
		return nil, apiErr
	}

	customers := repos.NewCustomerRepo()
	filters.AccountID, filters.Cursor, filters.Limit, filters.Includes = accountID, nil, customerExportPageSize, nil

	var rows []customerExportRow
	for {
		page, apiErr := customers.List(ctx, filters)
		if apiErr != nil {
			return nil, apiErr
		}

		contactsByCustomer, apiErr := customerContactsByAccount(ctx, customers, page.Items)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, customer := range page.Items {
			rows = append(rows, customerExportRow{
				customer:   customer,
				statusName: statusNames[customer.Status],
				contacts:   contactsByCustomer[customer.ID],
			})
		}

		// Past the cap is enough for the engine to refuse the export.
		if len(rows) > domain.ExportRowLimit || !page.PageInfo.HasNextPage || page.PageInfo.NextCursor == nil {
			return rows, nil
		}
		filters.Cursor = page.PageInfo.NextCursor
	}
}

// accountStatusNames names each account status code; the list row carries only the code.
func accountStatusNames(ctx context.Context, repos domain.RepoFactory) (map[constants.AccountStatusCode]string, *apierror.APIError) {
	statuses, apiErr := repos.NewAccountStatusRepo().List(ctx, domain.ListAccountStatusesParams{Limit: 100})
	if apiErr != nil {
		return nil, apiErr
	}
	names := make(map[constants.AccountStatusCode]string, len(statuses.AccountStatuses))
	for _, status := range statuses.AccountStatuses {
		names[constants.AccountStatusCode(status.Code)] = status.Name
	}
	return names, nil
}

func customerContactsByAccount(ctx context.Context, repo domain.CustomerRepo, customers []*domain.Customer) (map[string][]domain.CustomerContact, *apierror.APIError) {
	ids := make([]string, len(customers))
	for i, customer := range customers {
		ids[i] = customer.ID
	}
	contacts, apiErr := repo.ListContacts(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	byCustomer := make(map[string][]domain.CustomerContact, len(customers))
	for _, contact := range contacts {
		byCustomer[contact.CustomerAccountID] = append(byCustomer[contact.CustomerAccountID], contact)
	}
	return byCustomer, nil
}

func projectCustomerExportRow(row customerExportRow) excel.Row {
	c := row.customer
	return excel.Row{
		"customerNumber":   textutil.FormatAccountNumber(c.Number),
		"name":             c.Name,
		"email":            excel.Str(c.Email),
		"phone":            excel.Str(c.Phone),
		"status":           row.statusName,
		"customerGroup":    excel.Str(c.TypeGroupName),
		"salesRep":         excel.Str(c.DefaultSalesRepName),
		"priority":         excel.Str(c.DefaultPriorityName),
		"paymentTerm":      excel.Str(c.DefaultPaymentTermName),
		"shippingTerm":     excel.Str(c.DefaultShippingTermName),
		"carrier":          excel.Str(c.DefaultCarrierName),
		"defaultBilling":   customerAddressLine(c.BillToAddress),
		"defaultShipping":  customerAddressLine(c.ShipToAddress),
		"contacts":         customerContactsCell(row.contacts),
		"commissionExempt": yesNo(c.CommissionPolicy.ToBool()),
		"freightExempt":    yesNo(c.FreightPolicy.ToBool()),
		"parentAccount":    yesNo(c.ParentAccountID != nil),
		"note":             excel.Str(c.Note),
		"createdAt":        c.CreatedAt,
	}
}

// addressTail matches the last two comma-separated parts of an address line.
var addressTail = regexp.MustCompile(`, ([^,]*), ([^,]*)$`)

// customerAddressLine writes an address on one line as the dashboard did, its last two parts set off by spaces: "1 Main St, Springfield, IL 62701 US".
func customerAddressLine(address *domain.CustomerAddress) string {
	if address == nil || address.Geolocation == nil {
		return ""
	}
	g := address.Geolocation
	parts := make([]string, 0, 6)
	for _, part := range []string{excel.Str(g.StreetLine1), excel.Str(g.StreetLine2), excel.Str(g.Locality), excel.Str(g.State), excel.Str(g.PostalCode), g.Country} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return addressTail.ReplaceAllString(strings.Join(parts, ", "), " $1 $2")
}

// customerContactsCell writes "Name (email)" per member; one known by neither, such as a username-only portal user, is left out rather than written as an empty entry.
func customerContactsCell(contacts []domain.CustomerContact) string {
	entries := make([]string, 0, len(contacts))
	for _, contact := range contacts {
		name, email := excel.Str(contact.Name), excel.Str(contact.Email)
		switch {
		case name != "" && email != "":
			entries = append(entries, fmt.Sprintf("%s (%s)", name, email))
		case name != "" || email != "":
			entries = append(entries, name+email)
		}
	}
	return excel.JoinNames(entries)
}
