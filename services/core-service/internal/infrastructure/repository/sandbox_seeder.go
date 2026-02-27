package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/augno/api/services/core-service/internal/infrastructure/queries"
	"github.com/augno/api/shared/tracing"
)

var seederRepoTracer = tracing.GetTracer("core-service.sandbox_seeder")

type SandboxSeeder struct {
	db *sql.DB
}

func NewSandboxSeeder(db *sql.DB) *SandboxSeeder {
	return &SandboxSeeder{db: db}
}

func (r *SandboxSeeder) Seed(ctx context.Context, accountID string) error {
	ctx, span := seederRepoTracer.Start(ctx, "repository.sandbox_seeder.seed")
	defer span.End()

	// Verify the target account is a sandbox before seeding
	var accountTypeCode string
	err := r.db.QueryRowContext(ctx, "SELECT account_type_code FROM account WHERE id = ?", accountID).Scan(&accountTypeCode)
	if err == sql.ErrNoRows {
		return fmt.Errorf("account %s not found", accountID)
	}
	if err != nil {
		return fmt.Errorf("verify account type: %w", err)
	}
	if accountTypeCode != "sandbox" {
		return fmt.Errorf("refusing to seed non-sandbox account %s (type: %s)", accountID, accountTypeCode)
	}

	// Replace the placeholder with the actual account ID
	query := strings.ReplaceAll(queries.SandboxSeedSQL, "'@account_id'", fmt.Sprintf("'%s'", accountID))

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, stmt := range strings.Split(query, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec seed statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed tx: %w", err)
	}

	log.Printf("[seed] Successfully seeded sandbox account %s", accountID)
	return nil
}
