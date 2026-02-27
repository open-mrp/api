package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/augno/api/shared/tracing"
)

var purgeRepoTracer = tracing.GetTracer("core-service.purge_repository")

type tableColumn struct {
	Table  string
	Column string
}

var purgeTargets = []tableColumn{
	{"account_address", "account_id"},
	{"account_branding", "owner_account_id"},
	{"account_group", "owner_account_id"},
	{"account_integration", "account_id"},
	{"account_inventory_setting", "account_id"},
	{"account_portal", "owner_account_id"},
	{"account_price", "owner_account_id"},
	{"account_relation", "owner_account_id"},
	{"account_relation", "counterparty_account_id"},
	{"account_user", "account_id"},
	{"api_key", "owner_account_id"},
	{"attribute", "account_id"},
	{"batch", "account_id"},
	{"carrier", "account_id"},
	{"carrier_option", "account_id"},
	{"change_log", "account_id"},
	{"dc_location", "account_id"},
	{"delivery", "account_id"},
	{"department", "account_id"},
	{"edi_run", "account_id"},
	{"email_log", "account_id"},
	{"error_log", "account_id"},
	{"inventory_change_log", "account_id"},
	{"inventory_issue", "account_id"},
	{"inventory_log", "account_id"},
	{"inventory_receipt", "owner_account_id"},
	{"invoice", "account_id"},
	{"item", "account_id"},
	{"item_category", "account_id"},
	{"journal_posting", "account_id"},
	{"lot", "account_id"},
	{"order_discount", "account_id"},
	{"payment_term", "account_id"},
	{"pick", "account_id"},
	{"product_line", "account_id"},
	{"product_line_target", "account_id"},
	{"production_run", "account_id"},
	{"production_step", "account_id"},
	{"property", "account_id"},
	{"quantity_discount", "account_id"},
	{"receiving_order", "account_id"},
	{"registration_flow", "account_id"},
	{"role", "account_id"},
	{"sales_order", "owner_account_id"},
	{"scanning_station", "account_id"},
	{"settlement", "account_id"},
	{"shipment", "account_id"},
	{"shipping_case", "account_id"},
	{"shipping_term", "account_id"},
	{"storage_location", "account_id"},
	{"supplier_material", "owner_account_id"},
	{"sys_property", "account_id"},
	{"target", "account_id"},
	{"territory", "account_id"},
	{"transaction", "account_id"},
	{"unit", "account_id"},
	{"unit_group", "account_id"},
}

type PurgeRepo struct {
	db *sql.DB
}

func NewPurgeRepo(db *sql.DB) *PurgeRepo {
	return &PurgeRepo{db: db}
}

func (r *PurgeRepo) VerifyAccountIsSandboxOrDeleted(ctx context.Context, accountID string) error {
	ctx, span := purgeRepoTracer.Start(ctx, "repository.purge.verify_account_is_sandbox_or_deleted")
	defer span.End()

	var accountTypeCode string
	err := r.db.QueryRowContext(ctx, "SELECT account_type_code FROM account WHERE id = ?", accountID).Scan(&accountTypeCode)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("verify account type: %w", err)
	}
	if accountTypeCode != "sandbox" {
		return fmt.Errorf("refusing to purge non-sandbox account %s (type: %s)", accountID, accountTypeCode)
	}

	return nil
}

func (r *PurgeRepo) PurgeAccountData(ctx context.Context, accountID string) error {
	ctx, span := purgeRepoTracer.Start(ctx, "repository.purge.purge_account_data")
	defer span.End()

	for _, target := range purgeTargets {
		if err := r.deleteFromTable(ctx, target.Table, target.Column, accountID); err != nil {
			log.Printf("[purge] WARNING: Failed to purge %s.%s for account %s: %v", target.Table, target.Column, accountID, err)
			span.RecordError(err)
			return err
		}
	}

	return nil
}

func (r *PurgeRepo) deleteFromTable(ctx context.Context, table, column, accountID string) error {
	query := fmt.Sprintf("DELETE FROM `%s` WHERE `%s` = ?", table, column) // #nosec G201 - table/column names from hardcoded list
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for %s: %w", table, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, accountID); err != nil {
		return fmt.Errorf("delete from %s: %w", table, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit for %s: %w", table, err)
	}

	return nil
}
