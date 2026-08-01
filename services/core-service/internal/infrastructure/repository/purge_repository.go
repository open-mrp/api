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
	// Messaging substrate (in-app notifications, announcements, chat). announcement uses a nullable account_id (NULL = platform-wide), so this only removes account-scoped rows.
	{"announcement", "account_id"},
	{"conversation", "account_id"},
	{"conversation_dm_key", "account_id"},
	{"conversation_participant", "account_id"},
	{"message", "account_id"},
	{"message_attachment", "account_id"},
	{"messaging_block", "account_id"},
	{"notification", "account_id"},
	{"notification_preference", "account_id"},
	{"scheduled_message", "account_id"},
	// Production scheduling. The lookup tables (machine_downtime_reason, demand_override_type) are global, not account-scoped, so they are deliberately absent.
	{"account_production_schedule_setting", "account_id"},
	{"production_schedule_resource_setting", "account_id"},
	{"production_schedule_item_setting", "account_id"},
	{"production_shift", "account_id"},
	{"demand_override", "account_id"},
	{"machine_downtime_event", "account_id"},
	{"production_schedule", "account_id"},
	{"production_schedule_line", "account_id"},
	{"production_schedule_item_policy", "account_id"},
	{"production_schedule_deviation", "account_id"},
	{"production_schedule_derived_line", "account_id"},
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

// prePurgeJoinDeletes removes rows from tables that lack an account_id column but reference account-scoped parent rows. These must run before the main purgeTargets loop deletes the parent rows (item, unit_group, etc.).
var prePurgeJoinDeletes = []string{
	"DELETE p FROM product p JOIN item i ON p.item_id = i.id WHERE i.account_id = ?",
	"DELETE r FROM rate r JOIN item i ON r.id = i.unit_value_id WHERE i.account_id = ?",
	"DELETE r FROM rate r JOIN item i ON r.id = i.unit_cost_id WHERE i.account_id = ?",
	"DELETE r FROM rate r JOIN item i ON r.id = i.burn_rate_id WHERE i.account_id = ?",
	"DELETE ugu FROM unit_group_unit ugu JOIN unit_group ug ON ugu.unit_group_id = ug.id WHERE ug.account_id = ?",
	// Messaging receipt tables have no account_id column; delete via their account-scoped parents.
	"DELETE mr FROM message_receipt mr JOIN message m ON mr.message_id = m.id WHERE m.account_id = ?",
	"DELETE ar FROM announcement_receipt ar JOIN announcement a ON ar.announcement_id = a.id WHERE a.account_id = ?",
}

func (r *PurgeRepo) PurgeAccountData(ctx context.Context, accountID string) error {
	ctx, span := purgeRepoTracer.Start(ctx, "repository.purge.purge_account_data")
	defer span.End()

	for _, query := range prePurgeJoinDeletes {
		if err := r.execInTx(ctx, query, accountID); err != nil {
			log.Printf("[purge] WARNING: Failed pre-purge join delete for account %s: %v", accountID, err)
			span.RecordError(err)
			return err
		}
	}

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
	return r.execInTx(ctx, query, accountID)
}

func (r *PurgeRepo) execInTx(ctx context.Context, query, accountID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() // #nosec G104 - rollback on already-committed tx is a no-op

	if _, err := tx.ExecContext(ctx, query, accountID); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
