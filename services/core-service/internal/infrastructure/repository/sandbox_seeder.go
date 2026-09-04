package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/queries"
	"github.com/open-mrp/api/shared/tracing"
)

var seederRepoTracer = tracing.GetTracer("core-service.sandbox_seeder")

var (
	userVarSetRe = regexp.MustCompile(`(?i)^SET\s+@(\w+)\s*=\s*(.+)$`)
	userVarRefRe = regexp.MustCompile(`@(\w+)`)
)

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

	vars := map[string]string{}

	for stmt := range strings.SplitSeq(query, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if varName, expr, ok := parseUserVarSet(stmt); ok {
			resolvedExpr := substituteVars(expr, vars)
			value, resolveErr := resolveSetExpression(ctx, tx, resolvedExpr)
			if resolveErr != nil {
				return fmt.Errorf("resolve seed variable %s: %w", varName, resolveErr)
			}
			vars[varName] = value
			continue
		}

		resolvedStmt := substituteVars(stmt, vars)
		if _, err := tx.ExecContext(ctx, resolvedStmt); err != nil {
			return fmt.Errorf("exec seed statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed tx: %w", err)
	}

	log.Printf("[seed] Successfully seeded sandbox account %s", accountID)
	return nil
}

func parseUserVarSet(stmt string) (string, string, bool) {
	matches := userVarSetRe.FindStringSubmatch(stripLeadingLineComments(stmt))
	if len(matches) != 3 {
		return "", "", false
	}

	return matches[1], strings.TrimSpace(matches[2]), true
}

// stripLeadingLineComments drops the leading `--` lines a statement carries in from the text ahead of it. The seed is split on `;`, so a section header or a group note lands at the head of the following statement, and a SET sitting behind one would not be recognized as a variable assignment: it would go to the database as a raw `SET @var`, which is exactly what resolving variables in Go exists to avoid. The original statement is still what gets executed, so comments ahead of anything else are left alone.
func stripLeadingLineComments(stmt string) string {
	for {
		stmt = strings.TrimSpace(stmt)
		if !strings.HasPrefix(stmt, "--") {
			return stmt
		}
		_, rest, found := strings.Cut(stmt, "\n")
		if !found {
			return ""
		}
		stmt = rest
	}
}

func resolveSetExpression(ctx context.Context, tx *sql.Tx, expr string) (string, error) {
	query := expr
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		query = strings.TrimSpace(expr[1 : len(expr)-1])
	} else {
		query = "SELECT " + expr
	}

	var value sql.NullString
	if err := tx.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return "", err
	}

	if !value.Valid {
		return "", nil
	}

	return value.String, nil
}

func substituteVars(stmt string, vars map[string]string) string {
	return userVarRefRe.ReplaceAllStringFunc(stmt, func(token string) string {
		matches := userVarRefRe.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}

		value, ok := vars[matches[1]]
		if !ok {
			return token
		}

		escaped := strings.ReplaceAll(value, `'`, `''`)
		return "'" + escaped + "'"
	})
}
