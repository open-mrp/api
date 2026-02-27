package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// SAFETY: DO NOT REMOVE — these tests ensure the purge repository refuses
// to operate on non-sandbox accounts, protecting production data from
// accidental purge operations.

func TestSafety_VerifyAccountIsSandboxOrDeleted_RejectsNonSandbox(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT account_type_code FROM account WHERE id = ?").
		WithArgs("ac_production").
		WillReturnRows(sqlmock.NewRows([]string{"account_type_code"}).AddRow("company"))

	repo := NewPurgeRepo(db)
	verifyErr := repo.VerifyAccountIsSandboxOrDeleted(context.Background(), "ac_production")

	if verifyErr == nil {
		t.Fatal("expected error for non-sandbox account, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSafety_VerifyAccountIsSandboxOrDeleted_AllowsSandbox(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT account_type_code FROM account WHERE id = ?").
		WithArgs("ac_sandbox").
		WillReturnRows(sqlmock.NewRows([]string{"account_type_code"}).AddRow("sandbox"))

	repo := NewPurgeRepo(db)
	verifyErr := repo.VerifyAccountIsSandboxOrDeleted(context.Background(), "ac_sandbox")

	if verifyErr != nil {
		t.Fatalf("expected nil error for sandbox account, got %v", verifyErr)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSafety_VerifyAccountIsSandboxOrDeleted_AllowsDeletedAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT account_type_code FROM account WHERE id = ?").
		WithArgs("ac_deleted").
		WillReturnRows(sqlmock.NewRows([]string{"account_type_code"}))

	repo := NewPurgeRepo(db)
	verifyErr := repo.VerifyAccountIsSandboxOrDeleted(context.Background(), "ac_deleted")

	if verifyErr != nil {
		t.Fatalf("expected nil error for deleted (missing) account, got %v", verifyErr)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
