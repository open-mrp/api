package mediator

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/tracing"
)

var addressMedTracer = tracing.GetTracer("core-service.address_mediator")

type addressMedImpl struct {
	repos domain.RepoFactory
}

type AddressMedConfig struct {
	// Repos (required) is the repository factory the address writes and audit events go through.
	Repos domain.RepoFactory
}

func (c *AddressMedConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("address mediator: repos is required")
	}
	return nil
}

func NewAddressMed(config *AddressMedConfig) domain.AddressMed {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &addressMedImpl{repos: config.Repos}
}

// Create saves a new address in params.AccountID and records its audit event.
//
//  1. Trim the name; a blank name is a validation error on `name`.
//  2. Insert the geolocation, the address, and the address's link to the account.
//  3. Publish the address create audit event to the outbox.
func (m *addressMedImpl) Create(ctx context.Context, params domain.CreateAddressParams) (*domain.Address, *apierror.APIError) {
	ctx, span := addressMedTracer.Start(ctx, "mediator.address.create")
	defer span.End()

	name, apiErr := NormalizeAddressName(params.Name)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.Name = name

	created, apiErr := m.create(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return created, nil
}

// Update changes an address linked to params.AccountID and records its audit event. Fields the params leave unset keep their stored values.
//
//  1. Trim a given name; a blank name is a validation error on `name`.
//  2. Return not-found unless the address is linked to the account.
//  3. When a street, locality, state, postal code or country changes, update the geolocation in place, or move the address to a new one when other addresses share it.
//  4. Update the address, writing back the stored phone, email and receiving calendar where the params leave them unset.
//  5. Publish the address update audit event to the outbox.
func (m *addressMedImpl) Update(ctx context.Context, params domain.UpdateAddressParams) (*domain.Address, *apierror.APIError) {
	ctx, span := addressMedTracer.Start(ctx, "mediator.address.update")
	defer span.End()

	name, apiErr := NormalizeOptionalAddressName(params.Name)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	params.Name = name

	inAccount, apiErr := m.repos.NewAddressRepo().IsInAccount(ctx, params.AccountID, params.AddressID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !inAccount {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Address not found."))
	}

	updated, apiErr := m.update(ctx, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return updated, nil
}

func (m *addressMedImpl) create(ctx context.Context, params domain.CreateAddressParams) (*domain.Address, *apierror.APIError) {
	addressID, apiErr := id.GenID(id.AddressIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	geolocationID, apiErr := id.GenID(id.GeolocationIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	accountAddressID, apiErr := id.GenID(id.AccountAddressIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	created, apiErr := m.repos.NewAddressRepo().Create(ctx, addressID, geolocationID, accountAddressID, params)
	if apiErr != nil {
		return nil, apiErr
	}

	if apiErr := audit.NewPublisher().Publish(ctx, m.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionCreate,
		ResourceType: constants.ObjectTypeAddress,
		ResourceID:   created.ID,
		Changes:      audit.ComputeChanges(nil, created),
	}); apiErr != nil {
		return nil, apiErr
	}
	return created, nil
}

func (m *addressMedImpl) update(ctx context.Context, params domain.UpdateAddressParams) (*domain.Address, *apierror.APIError) {
	repo := m.repos.NewAddressRepo()

	existing, apiErr := repo.Get(ctx, domain.GetAddressParams{
		AccountID: params.AccountID,
		AddressID: params.AddressID,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	if apiErr := m.updateGeolocation(ctx, existing, params); apiErr != nil {
		return nil, apiErr
	}

	params.Phone = params.Phone.BackfillUnsetPtr(existing.Phone)
	params.Email = params.Email.BackfillUnsetPtr(existing.Email)
	params.ReceiveCalendarID = params.ReceiveCalendarID.BackfillUnsetPtr(existing.ReceiveCalendarID)

	updated, apiErr := repo.Update(ctx, params)
	if apiErr != nil {
		return nil, apiErr
	}

	if apiErr := audit.NewPublisher().Publish(ctx, m.repos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeAddress,
		ResourceID:   updated.ID,
		Changes:      audit.ComputeChanges(existing, updated),
	}); apiErr != nil {
		return nil, apiErr
	}
	return updated, nil
}

// updateGeolocation applies the street-level fields of an update. A geolocation other addresses share is left alone and the address moves to a copy, so the edit reaches only this address; any geo change drops the stored place match.
func (m *addressMedImpl) updateGeolocation(ctx context.Context, existing *domain.Address, params domain.UpdateAddressParams) *apierror.APIError {
	repo := m.repos.NewAddressRepo()
	streetLine2 := params.StreetLine2

	if !coreGeoChanged(existing.Geolocation, params) {
		if !streetLine2.WasProvided() {
			return nil
		}
		geoID, apiErr := repo.GetGeolocationIDByAddressID(ctx, params.AddressID)
		if apiErr != nil {
			return apiErr
		}
		return repo.UpdateGeolocation(ctx, geoID, domain.UpdateAddressParams{StreetLine2: streetLine2})
	}

	geoID, apiErr := repo.GetGeolocationIDByAddressID(ctx, params.AddressID)
	if apiErr != nil {
		return apiErr
	}
	sharedCount, apiErr := repo.GetGeolocationSharedCount(ctx, geoID)
	if apiErr != nil {
		return apiErr
	}

	if sharedCount <= 1 {
		inPlace := params
		inPlace.StreetLine2 = streetLine2.BackfillUnsetPtr(existing.Geolocation.StreetLine2)
		return repo.UpdateGeolocation(ctx, geoID, inPlace)
	}

	newGeoID, apiErr := id.GenID(id.GeolocationIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	if apiErr := repo.CreateGeolocation(ctx, newGeoID, domain.CreateAddressParams{
		StreetLine1: coalesceStringPtr(params.StreetLine1, existing.Geolocation.StreetLine1),
		StreetLine2: streetLine2.StringPtrAfterBackfill(existing.Geolocation.StreetLine2),
		Locality:    coalesceStringPtr(params.Locality, existing.Geolocation.Locality),
		State:       coalesceStringPtr(params.State, existing.Geolocation.State),
		PostalCode:  coalesceStringPtr(params.PostalCode, existing.Geolocation.PostalCode),
		Country:     coalesceString(params.Country, existing.Geolocation.Country),
	}); apiErr != nil {
		return apiErr
	}
	return repo.RelinkGeolocation(ctx, params.AddressID, newGeoID)
}

// coreGeoChanged reports whether the update moves the address: a new street, locality, state, postal code or country.
func coreGeoChanged(geo *domain.Geolocation, params domain.UpdateAddressParams) bool {
	return stringPtrChanged(params.StreetLine1, geo.StreetLine1) ||
		stringPtrChanged(params.Locality, geo.Locality) ||
		stringPtrChanged(params.State, geo.State) ||
		stringPtrChanged(params.PostalCode, geo.PostalCode) ||
		(params.Country != nil && *params.Country != geo.Country)
}

func stringPtrChanged(update, stored *string) bool {
	return update != nil && (stored == nil || *update != *stored)
}

func coalesceStringPtr(update, existing *string) *string {
	if update != nil {
		return update
	}
	return existing
}

func coalesceString(update *string, existing string) string {
	if update != nil {
		return *update
	}
	return existing
}

// NormalizeAddressName trims an address name; a blank one is a validation error on `name`.
func NormalizeAddressName(name string) (string, *apierror.APIError) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", apierror.NewValidationErrorWithParam("Address name is required.", "name")
	}
	return trimmed, nil
}

// NormalizeOptionalAddressName is NormalizeAddressName for an update, where nil leaves the name as stored.
func NormalizeOptionalAddressName(name *string) (*string, *apierror.APIError) {
	if name == nil {
		return nil, nil
	}
	normalized, apiErr := NormalizeAddressName(*name)
	if apiErr != nil {
		return nil, apiErr
	}
	return &normalized, nil
}
