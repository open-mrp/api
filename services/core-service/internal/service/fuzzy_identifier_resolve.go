package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Fuzzy reference resolution. A request may point at an existing entity by any one of
// several identifiers — an id, a name, an abbreviation, a SKU — and the server resolves
// it. Resolution precedence is always id first, then the human-readable identifiers.
//
// Resolving by a non-id identifier is only sound because every entity referenced this
// way carries a UNIQUE(account_id, <identifier>): department, scanning_station,
// item_category, product_line, unit_group, property and storage_location on name; unit
// on BOTH name and abbreviation; item on sku. Do NOT add a fuzzy identifier for an
// entity that lacks that unique key — the lookup would be ambiguous and could silently
// bind the wrong row. (machine.serial_number, for example, has no unique key today.)
//
// Each resolver batches its lookups: one query for the id-identifiers, one for the
// name/abbreviation/SKU-identifiers. Callers then resolve each identifier from memory, so error
// reporting — fail-fast with a row-indexed param, or collected across all rows — stays
// with the caller, which is what differs between endpoints.

// addKey records key in set, returning true if it was newly added.
func addKey(set map[string]struct{}, key string) bool {
	if _, ok := set[key]; ok {
		return false
	}
	set[key] = struct{}{}
	return true
}

// unitIdentifierLabel returns the identifier a unit identifier will actually be resolved by, for
// error messages.
func unitIdentifierLabel(identifier domain.UnitIdentifier) string {
	switch {
	case identifier.ID != "":
		return identifier.ID
	case identifier.Name != "":
		return identifier.Name
	case identifier.Abbreviation != "":
		return identifier.Abbreviation
	default:
		return ""
	}
}

// itemIdentifierLabel returns the identifier an item identifier will actually be resolved by.
func itemIdentifierLabel(identifier domain.ItemIdentifier) string {
	if identifier.ID != "" {
		return identifier.ID
	}
	return identifier.SKU
}

// objectIdentifierLabel returns the identifier a named identifier will actually be resolved by.
func objectIdentifierLabel(id, name string) string {
	if id != "" {
		return id
	}
	return name
}

// --- Units: id, then name, then abbreviation ---

// resolvedUnit is a unit identifier resolved to its ID, carrying the dimension code so
// cost-rate validation needs no second lookup.
type resolvedUnit struct {
	ID            string
	DimensionCode string
}

type unitIdentifierResolver struct {
	byID   map[string]resolvedUnit
	byName map[string]resolvedUnit
	byAbbr map[string]resolvedUnit
}

// newUnitIdentifierResolver batch-loads every unit the identifiers point at: one GetByIDs for the
// id-identifiers and one FindByAbbreviationsOrNames for the name/abbreviation identifiers.
func newUnitIdentifierResolver(ctx context.Context, repos domain.RepoFactory, accountID string, identifiers []domain.UnitIdentifier) (*unitIdentifierResolver, *apierror.APIError) {
	idSet, nameSet, abbrSet := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	var ids, names, abbrs []string
	for _, identifier := range identifiers {
		switch {
		case identifier.ID != "":
			if addKey(idSet, identifier.ID) {
				ids = append(ids, identifier.ID)
			}
		case identifier.Name != "":
			if addKey(nameSet, strings.ToLower(identifier.Name)) {
				names = append(names, identifier.Name)
			}
		case identifier.Abbreviation != "":
			if addKey(abbrSet, strings.ToLower(identifier.Abbreviation)) {
				abbrs = append(abbrs, identifier.Abbreviation)
			}
		}
	}

	r := &unitIdentifierResolver{
		byID:   map[string]resolvedUnit{},
		byName: map[string]resolvedUnit{},
		byAbbr: map[string]resolvedUnit{},
	}
	repo := repos.NewUnitRepo()
	if len(ids) > 0 {
		units, apiErr := repo.GetByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, u := range units {
			r.byID[u.ID] = resolvedUnit{ID: u.ID, DimensionCode: u.UnitDimensionCode}
		}
	}
	if len(names) > 0 || len(abbrs) > 0 {
		units, apiErr := repo.FindByAbbreviationsOrNames(ctx, accountID, abbrs, names)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, u := range units {
			resolved := resolvedUnit{ID: u.ID, DimensionCode: u.UnitDimensionCode}
			r.byName[strings.ToLower(u.Name)] = resolved
			r.byAbbr[strings.ToLower(u.Abbreviation)] = resolved
		}
	}
	return r, nil
}

// resolve returns the unit the identifier points at. ok is false when the identifier is empty or the
// unit was not found in the account.
func (r *unitIdentifierResolver) resolve(identifier domain.UnitIdentifier) (resolvedUnit, bool) {
	switch {
	case identifier.ID != "":
		u, ok := r.byID[identifier.ID]
		return u, ok
	case identifier.Name != "":
		u, ok := r.byName[strings.ToLower(identifier.Name)]
		return u, ok
	case identifier.Abbreviation != "":
		u, ok := r.byAbbr[strings.ToLower(identifier.Abbreviation)]
		return u, ok
	default:
		return resolvedUnit{}, false
	}
}

// resolveOrError resolves the identifier, returning a validation error at param when it is
// empty or unknown.
func (r *unitIdentifierResolver) resolveOrError(identifier domain.UnitIdentifier, param string) (resolvedUnit, *apierror.APIError) {
	if identifier == (domain.UnitIdentifier{}) {
		return resolvedUnit{}, apierror.NewValidationErrorWithParam("A unit id, name, or abbreviation is required.", param)
	}
	u, ok := r.resolve(identifier)
	if !ok {
		return resolvedUnit{}, apierror.NewValidationErrorWithParam(
			fmt.Sprintf("Unit %q was not found.", unitIdentifierLabel(identifier)), param)
	}
	return u, nil
}

// --- Items: id, then SKU ---

type itemIdentifierResolver struct {
	validIDs map[string]struct{}
	idBySKU  map[string]string
}

// newItemIdentifierResolver batch-loads every item the identifiers point at: one GetByIDs for the
// id-identifiers and one FetchItemsBySKU for the SKU-identifiers.
func newItemIdentifierResolver(ctx context.Context, repos domain.RepoFactory, accountID string, identifiers []domain.ItemIdentifier) (*itemIdentifierResolver, *apierror.APIError) {
	idSet, skuSet := map[string]struct{}{}, map[string]struct{}{}
	var ids, skus []string
	for _, identifier := range identifiers {
		switch {
		case identifier.ID != "":
			if addKey(idSet, identifier.ID) {
				ids = append(ids, identifier.ID)
			}
		case identifier.SKU != "":
			if addKey(skuSet, strings.ToLower(identifier.SKU)) {
				skus = append(skus, identifier.SKU)
			}
		}
	}

	r := &itemIdentifierResolver{validIDs: map[string]struct{}{}, idBySKU: map[string]string{}}
	repo := repos.NewItemRepo()
	if len(ids) > 0 {
		items, apiErr := repo.GetByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, item := range items {
			r.validIDs[item.ID] = struct{}{}
		}
	}
	if len(skus) > 0 {
		items, apiErr := repo.FetchItemsBySKU(ctx, accountID, skus)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, item := range items {
			r.idBySKU[strings.ToLower(item.SKU)] = item.ItemID
		}
	}
	return r, nil
}

// resolve returns the item ID the identifier points at.
func (r *itemIdentifierResolver) resolve(identifier domain.ItemIdentifier) (string, bool) {
	switch {
	case identifier.ID != "":
		_, ok := r.validIDs[identifier.ID]
		return identifier.ID, ok
	case identifier.SKU != "":
		id, ok := r.idBySKU[strings.ToLower(identifier.SKU)]
		return id, ok
	default:
		return "", false
	}
}

// resolveOrError resolves the identifier, returning a validation error at param when it is
// empty or unknown.
func (r *itemIdentifierResolver) resolveOrError(identifier domain.ItemIdentifier, param string) (string, *apierror.APIError) {
	if identifier == (domain.ItemIdentifier{}) {
		return "", apierror.NewValidationErrorWithParam("An item id or SKU is required.", param)
	}
	id, ok := r.resolve(identifier)
	if !ok {
		return "", apierror.NewValidationErrorWithParam(
			fmt.Sprintf("Item %q was not found.", itemIdentifierLabel(identifier)), param)
	}
	return id, nil
}

// --- Named entities (department, scanning station, location, category, …): id, then name ---

type objectIdentifierResolver struct {
	validIDs map[string]struct{}
	idByName map[string]string
	entity   string
}

// newObjectIdentifierResolver batch-loads every entity the identifiers point at, using the repo's
// GetByIDs and FindByNames. idOf/nameOf project the repo's row type; entity names the
// thing in error messages ("department", "location", …).
func newObjectIdentifierResolver[T any](
	ctx context.Context,
	accountID, entity string,
	identifiers []domain.ObjectIdentifier,
	getByIDs func(context.Context, string, []string) ([]T, *apierror.APIError),
	findByNames func(context.Context, string, []string) ([]T, *apierror.APIError),
	idOf func(T) string,
	nameOf func(T) string,
) (*objectIdentifierResolver, *apierror.APIError) {
	idSet, nameSet := map[string]struct{}{}, map[string]struct{}{}
	var ids, names []string
	for _, identifier := range identifiers {
		switch {
		case identifier.ID != "":
			if addKey(idSet, identifier.ID) {
				ids = append(ids, identifier.ID)
			}
		case identifier.Name != "":
			if addKey(nameSet, strings.ToLower(identifier.Name)) {
				names = append(names, identifier.Name)
			}
		}
	}

	r := &objectIdentifierResolver{validIDs: map[string]struct{}{}, idByName: map[string]string{}, entity: entity}
	if len(ids) > 0 {
		found, apiErr := getByIDs(ctx, accountID, ids)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, e := range found {
			r.validIDs[idOf(e)] = struct{}{}
		}
	}
	if len(names) > 0 {
		found, apiErr := findByNames(ctx, accountID, names)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, e := range found {
			r.idByName[strings.ToLower(nameOf(e))] = idOf(e)
		}
	}
	return r, nil
}

// resolve returns the entity ID the identifier points at.
func (r *objectIdentifierResolver) resolve(identifier domain.ObjectIdentifier) (string, bool) {
	switch {
	case identifier.ID != "":
		_, ok := r.validIDs[identifier.ID]
		return identifier.ID, ok
	case identifier.Name != "":
		id, ok := r.idByName[strings.ToLower(identifier.Name)]
		return id, ok
	default:
		return "", false
	}
}

// resolveOrError resolves the identifier, returning a validation error at param when it is
// empty or unknown.
func (r *objectIdentifierResolver) resolveOrError(identifier domain.ObjectIdentifier, param string) (string, *apierror.APIError) {
	if identifier == (domain.ObjectIdentifier{}) {
		return "", apierror.NewValidationErrorWithParam(
			fmt.Sprintf("A %s id or name is required.", r.entity), param)
	}
	id, ok := r.resolve(identifier)
	if !ok {
		return "", apierror.NewValidationErrorWithParam(
			fmt.Sprintf("%s %q was not found.", capitalize(r.entity), objectIdentifierLabel(identifier.ID, identifier.Name)), param)
	}
	return id, nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// resolveObjectIdentifiers is the row-aligned convenience wrapper used by the bulk upserts that
// report ALL unresolved identifiers in one error (departments, machines, scanning stations)
// rather than failing fast. identifiers is aligned with the request rows; a nil entry means the
// row omitted the reference. rowsField and refField build the row-indexed param
// ("departments[2].location"), and entityPlural heads the message.
func resolveObjectIdentifiers[T any](
	txCtx context.Context,
	accountID string,
	rowsField, refField, entity, entityPlural string,
	identifiers []*domain.ObjectIdentifier,
	getByIDs func(context.Context, string, []string) ([]T, *apierror.APIError),
	findByNames func(context.Context, string, []string) ([]T, *apierror.APIError),
	idOf func(T) string,
	nameOf func(T) string,
) (map[domain.ObjectIdentifier]string, *apierror.APIError) {
	present := make([]domain.ObjectIdentifier, 0, len(identifiers))
	for _, identifier := range identifiers {
		if identifier != nil {
			present = append(present, *identifier)
		}
	}
	resolver, apiErr := newObjectIdentifierResolver(txCtx, accountID, entity, present, getByIDs, findByNames, idOf, nameOf)
	if apiErr != nil {
		return nil, apiErr
	}

	idByIdentifier := make(map[domain.ObjectIdentifier]string, len(present))
	var rowErrs apierror.RowErrors
	for i, identifier := range identifiers {
		if identifier == nil {
			continue
		}
		param := fmt.Sprintf("%s[%d].%s", rowsField, i, refField)
		id, apiErr := resolver.resolveOrError(*identifier, param)
		if apiErr != nil {
			rowErrs.AddValidation(i, param, strings.TrimSuffix(apiErr.PublicMessage, "."))
			continue
		}
		idByIdentifier[*identifier] = id
	}
	if apiErr := rowErrs.Summary(entityPlural); apiErr != nil {
		return nil, apiErr
	}
	return idByIdentifier, nil
}
