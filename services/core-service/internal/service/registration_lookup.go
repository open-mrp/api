package service

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"go.opentelemetry.io/otel/trace"
)

// existingCustomerNotFoundMessage is the same whether the number is unknown or the ZIP is wrong, so a guess learns nothing about which part missed.
const existingCustomerNotFoundMessage = "We couldn't find an account with that number and ZIP code."

// findExistingCustomer resolves the customer an existing-customer registration joins. The number alone links nobody: the ZIP must also match the customer's default billing or shipping address.
func findExistingCustomer(ctx context.Context, repo domain.CustomerRegistrationRepo, ownerAccountID string, params domain.RegisterCustomerParams) (string, *apierror.APIError) {
	lookup := params.Lookup
	if lookup.Type == "" {
		lookup.Type = constants.PortalRegistrationLookupTypeCustomerNumber
	}
	if !lookup.Type.IsValid() {
		return "", apierror.NewFieldError("lookup_type", apierror.CodeInvalidFormat, "Lookup type must be one of: "+strings.Join(lookup.Type.EnumValues(), ", ")+".")
	}
	number := strings.TrimSpace(lookup.Number)
	if number == "" && lookup.Type == constants.PortalRegistrationLookupTypeCustomerNumber && params.CustomerData.Number != nil {
		number = strings.TrimSpace(*params.CustomerData.Number)
	}
	if number == "" {
		return "", apierror.NewFieldError("lookup_number", apierror.CodeInvalidFormat, "Enter your customer, order, or invoice number.")
	}
	postalCode := normalizePostalCode(lookup.PostalCode)
	if postalCode == "" {
		return "", apierror.NewFieldError("lookup_postal_code", apierror.CodeInvalidFormat, "Enter the ZIP code on your account.")
	}

	var customerAccountID string
	var apiErr *apierror.APIError
	switch lookup.Type {
	case constants.PortalRegistrationLookupTypeSalesOrderNumber:
		customerAccountID, apiErr = repo.FindCustomerAccountBySalesOrderNumber(ctx, ownerAccountID, number)
	case constants.PortalRegistrationLookupTypeInvoiceNumber:
		customerAccountID, apiErr = repo.FindCustomerAccountByInvoiceNumber(ctx, ownerAccountID, number)
	default:
		customerAccountID, apiErr = repo.FindCustomerAccountByExternalNumber(ctx, ownerAccountID, number)
	}
	if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
		return "", apierror.NewNotFoundError(existingCustomerNotFoundMessage)
	}
	if apiErr != nil {
		return "", apiErr
	}

	codes, apiErr := repo.GetCustomerDefaultPostalCodes(ctx, ownerAccountID, customerAccountID)
	if apierror.HasCode(apiErr, apierror.CodeResourceNotFound) {
		return "", apierror.NewNotFoundError(existingCustomerNotFoundMessage)
	}
	if apiErr != nil {
		return "", apiErr
	}
	if !slices.ContainsFunc(codes, func(code string) bool { return normalizePostalCode(code) == postalCode }) {
		return "", apierror.NewNotFoundError(existingCustomerNotFoundMessage)
	}
	return customerAccountID, nil
}

var usZipPattern = regexp.MustCompile(`^(\d{5})(\d{4})?$`)

// normalizePostalCode compares a US ZIP by its first five digits, so 68137 and 68137-3211 match; other codes compare without spaces or dashes.
func normalizePostalCode(code string) string {
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' {
			return -1
		}
		return unicode.ToUpper(r)
	}, code)
	if m := usZipPattern.FindStringSubmatch(compact); m != nil {
		return m[1]
	}
	return compact
}

// freeEmailDomains are mailbox providers, which say nothing about which company a registrant works for.
var freeEmailDomains = map[string]bool{
	"aol.com": true, "comcast.net": true, "gmail.com": true, "googlemail.com": true, "hotmail.com": true,
	"icloud.com": true, "live.com": true, "mac.com": true, "me.com": true, "msn.com": true,
	"outlook.com": true, "proton.me": true, "protonmail.com": true, "yahoo.com": true, "ymail.com": true,
	"att.net": true, "sbcglobal.net": true, "verizon.net": true, "charter.net": true, "cox.net": true,
}

// findPossibleDuplicateCustomers names the seller's customers that a new registration may duplicate: one whose default address is the registrant's, or one with a user at the registrant's company email domain. It is advice for customer service, so a failed lookup returns what it has rather than failing the registration.
func findPossibleDuplicateCustomers(ctx context.Context, repo domain.CustomerRegistrationRepo, ownerAccountID, email string, address *domain.CustomerRegistrationAddress) []messaging.PossibleDuplicateCustomer {
	var found []messaging.PossibleDuplicateCustomer
	add := func(match domain.CustomerMatch, reason string) {
		for i := range found {
			if found[i].CustomerAccountID == match.AccountID {
				if !slices.Contains(found[i].Reasons, reason) {
					found[i].Reasons = append(found[i].Reasons, reason)
				}
				return
			}
		}
		found = append(found, messaging.PossibleDuplicateCustomer{
			CustomerAccountID: match.AccountID,
			CustomerName:      match.Name,
			CustomerNumber:    match.Number,
			Reasons:           []string{reason},
		})
	}

	if address != nil {
		postal := normalizePostalCode(address.PostalCode)
		street := streetKey(address.StreetLine1)
		if len(postal) >= 5 && street != "" {
			rows, apiErr := repo.ListCustomerDefaultAddressesInPostalArea(ctx, ownerAccountID, postal[:5])
			if apiErr != nil {
				_ = tracing.Trace(trace.SpanFromContext(ctx), apiErr)
			}
			for _, row := range rows {
				if sameAddress(row.BillingStreetLine1, row.BillingPostalCode, street, postal) ||
					sameAddress(row.ShippingStreetLine1, row.ShippingPostalCode, street, postal) {
					add(row.CustomerMatch, "same address")
				}
			}
		}
	}

	if at := strings.LastIndex(email, "@"); at >= 0 {
		domainName := strings.ToLower(strings.TrimSpace(email[at+1:]))
		if domainName != "" && !freeEmailDomains[domainName] {
			matches, apiErr := repo.ListCustomersWithUserEmailDomain(ctx, ownerAccountID, domainName)
			if apiErr != nil {
				_ = tracing.Trace(trace.SpanFromContext(ctx), apiErr)
			}
			for _, match := range matches {
				add(match, "same email domain")
			}
		}
	}
	return found
}

func sameAddress(streetLine1, postalCode *string, street, postal string) bool {
	return streetLine1 != nil && postalCode != nil && streetKey(*streetLine1) == street && normalizePostalCode(*postalCode) == postal
}

// streetKey reduces a street line to its house number and the first word of the street name, so "12741 Q St" and "12741 Q Street" agree.
func streetKey(line string) string {
	words := strings.FieldsFunc(strings.ToLower(line), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(words) < 2 {
		return ""
	}
	return words[0] + " " + words[1]
}
