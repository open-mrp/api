package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"
)

func TestNormalizePostalCode(t *testing.T) {
	for in, want := range map[string]string{
		"62701":      "62701",
		"62701-3211": "62701",
		"627013211":  "62701",
		" 62701 ":    "62701",
		"k1a 0b1":    "K1A0B1",
		"SW1A-1AA":   "SW1A1AA",
		"":           "",
	} {
		assert.Equal(t, want, normalizePostalCode(in), in)
	}
}

func TestStreetKey(t *testing.T) {
	assert.Equal(t, streetKey("4410 Main St"), streetKey("4410 Main Street"))
	assert.Equal(t, streetKey("4410 Main St"), streetKey("4410 Main Street, Springfield, IL 62701 US"))
	assert.NotEqual(t, streetKey("4410 Main St"), streetKey("4412 Main Street"))
	assert.Equal(t, "", streetKey("Main"))
}

func TestFindExistingCustomer(t *testing.T) {
	ctx := context.Background()
	const owner, customer = "ac_seller", "ac_buyer"

	newRepo := func(t *testing.T) *repositorymock.MockCustomerRegistrationRepo {
		return repositorymock.NewMockCustomerRegistrationRepo(gomock.NewController(t))
	}
	lookup := func(kind constants.PortalRegistrationLookupType, number, postal string) domain.RegisterCustomerParams {
		return domain.RegisterCustomerParams{Lookup: domain.ExistingCustomerLookup{Type: kind, Number: number, PostalCode: postal}}
	}

	t.Run("each number type finds the customer and the ZIP confirms it", func(t *testing.T) {
		for kind, expect := range map[constants.PortalRegistrationLookupType]func(*repositorymock.MockCustomerRegistrationRepo){
			constants.PortalRegistrationLookupTypeCustomerNumber: func(r *repositorymock.MockCustomerRegistrationRepo) {
				r.EXPECT().FindCustomerAccountByExternalNumber(ctx, owner, "104233").Return(customer, nil)
			},
			constants.PortalRegistrationLookupTypeSalesOrderNumber: func(r *repositorymock.MockCustomerRegistrationRepo) {
				r.EXPECT().FindCustomerAccountBySalesOrderNumber(ctx, owner, "104233").Return(customer, nil)
			},
			constants.PortalRegistrationLookupTypeInvoiceNumber: func(r *repositorymock.MockCustomerRegistrationRepo) {
				r.EXPECT().FindCustomerAccountByInvoiceNumber(ctx, owner, "104233").Return(customer, nil)
			},
		} {
			repo := newRepo(t)
			expect(repo)
			repo.EXPECT().GetCustomerDefaultPostalCodes(ctx, owner, customer).Return([]string{"62701-3211"}, nil)
			got, apiErr := findExistingCustomer(ctx, repo, owner, lookup(kind, " 104233 ", "62701"))
			require.Nil(t, apiErr, kind)
			assert.Equal(t, customer, got, kind)
		}
	})

	t.Run("a wrong ZIP reads like an unknown number", func(t *testing.T) {
		repo := newRepo(t)
		repo.EXPECT().FindCustomerAccountByExternalNumber(ctx, owner, "104233").Return(customer, nil)
		repo.EXPECT().GetCustomerDefaultPostalCodes(ctx, owner, customer).Return([]string{"62701"}, nil)
		_, wrongZip := findExistingCustomer(ctx, repo, owner, lookup("", "104233", "62704"))

		repo = newRepo(t)
		repo.EXPECT().FindCustomerAccountByExternalNumber(ctx, owner, "999").Return("", apierror.NewResourceNotFoundError("Customer not found."))
		_, unknown := findExistingCustomer(ctx, repo, owner, lookup("", "999", "62701"))

		require.NotNil(t, wrongZip)
		require.NotNil(t, unknown)
		assert.True(t, apierror.IsNotFound(wrongZip))
		assert.Equal(t, unknown.PublicMessage, wrongZip.PublicMessage)
	})

	t.Run("the legacy customer number field still finds the customer", func(t *testing.T) {
		repo := newRepo(t)
		repo.EXPECT().FindCustomerAccountByExternalNumber(ctx, owner, "104233").Return(customer, nil)
		repo.EXPECT().GetCustomerDefaultPostalCodes(ctx, owner, customer).Return([]string{"62701"}, nil)
		params := lookup("", "", "62701")
		params.CustomerData.Number = ptr("104233")
		got, apiErr := findExistingCustomer(ctx, repo, owner, params)
		require.Nil(t, apiErr)
		assert.Equal(t, customer, got)
	})

	t.Run("number and ZIP are both required", func(t *testing.T) {
		_, apiErr := findExistingCustomer(ctx, newRepo(t), owner, lookup("", "104233", ""))
		require.NotNil(t, apiErr)
		assert.Equal(t, "lookup_postal_code", apiErr.Param)

		_, apiErr = findExistingCustomer(ctx, newRepo(t), owner, lookup(constants.PortalRegistrationLookupTypeInvoiceNumber, " ", "62701"))
		require.NotNil(t, apiErr)
		assert.Equal(t, "lookup_number", apiErr.Param)

		_, apiErr = findExistingCustomer(ctx, newRepo(t), owner, lookup("po_number", "1", "62701"))
		require.NotNil(t, apiErr)
		assert.Equal(t, "lookup_type", apiErr.Param)
	})
}

func TestFindPossibleDuplicateCustomers(t *testing.T) {
	ctx := context.Background()
	northside := domain.CustomerMatch{AccountID: "ac_northside", Name: "Northside Pharmacy / Main", Number: "104233"}
	address := &domain.CustomerRegistrationAddress{StreetLine1: "4410 Main St", PostalCode: "62701-3211"}

	t.Run("same address and same company domain merge into one match", func(t *testing.T) {
		repo := repositorymock.NewMockCustomerRegistrationRepo(gomock.NewController(t))
		repo.EXPECT().ListCustomerDefaultAddressesInPostalArea(ctx, "ac_seller", "62701").Return([]domain.CustomerDefaultAddresses{
			{CustomerMatch: northside, BillingStreetLine1: ptr("4410 Main Street"), BillingPostalCode: ptr("62701")},
			{CustomerMatch: domain.CustomerMatch{AccountID: "ac_other"}, BillingStreetLine1: ptr("900 Elm Street"), BillingPostalCode: ptr("62701")},
		}, nil)
		repo.EXPECT().ListCustomersWithUserEmailDomain(ctx, "ac_seller", "northside.com").Return([]domain.CustomerMatch{northside}, nil)

		got := findPossibleDuplicateCustomers(ctx, repo, "ac_seller", "Buyer@Northside.com", address)
		assert.Equal(t, []messaging.PossibleDuplicateCustomer{{
			CustomerAccountID: "ac_northside",
			CustomerName:      "Northside Pharmacy / Main",
			CustomerNumber:    "104233",
			Reasons:           []string{"same address", "same email domain"},
		}}, got)
	})

	t.Run("a free mailbox domain is not a signal", func(t *testing.T) {
		repo := repositorymock.NewMockCustomerRegistrationRepo(gomock.NewController(t))
		repo.EXPECT().ListCustomerDefaultAddressesInPostalArea(ctx, "ac_seller", "62701").Return(nil, nil)
		assert.Empty(t, findPossibleDuplicateCustomers(ctx, repo, "ac_seller", "someone@gmail.com", address))
	})

	t.Run("a failed lookup does not fail the registration", func(t *testing.T) {
		repo := repositorymock.NewMockCustomerRegistrationRepo(gomock.NewController(t))
		repo.EXPECT().ListCustomerDefaultAddressesInPostalArea(ctx, "ac_seller", "62701").Return(nil, apierror.NewInternalError(assert.AnError, "boom"))
		repo.EXPECT().ListCustomersWithUserEmailDomain(ctx, "ac_seller", "northside.com").Return([]domain.CustomerMatch{northside}, nil)
		got := findPossibleDuplicateCustomers(ctx, repo, "ac_seller", "a@northside.com", address)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"same email domain"}, got[0].Reasons)
	})
}
