package repository

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
)

func setupMockRepo(t *testing.T) (*userRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}
	cleanup := func() { _ = sqlDB.Close() }
	return &userRepository{db: gdb}, mock, cleanup
}

func assertMockExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestCreate_Success(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs("u1", "u1@example.com", "google", "pid1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	user := &domain.User{Username: "u1", Email: "u1@example.com", Provider: "google", ProviderID: "pid1"}
	if err := repo.Create(user); err != nil {
		t.Fatalf("expected create success, got %v", err)
	}
	assertMockExpectations(t, mock)
}

func TestCreate_DBError(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs("u1", "u1@example.com", "", "", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	user := &domain.User{Username: "u1", Email: "u1@example.com"}
	if err := repo.Create(user); err == nil || !strings.Contains(err.Error(), "failed to create user") {
		t.Fatalf("expected create db error, got %v", err)
	}
	assertMockExpectations(t, mock)
}

func TestGetByUsername_SuccessAndNotFound(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("alice", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "provider", "provider_id"}).AddRow(1, "alice", "alice@example.com", "", ""))

	user, err := repo.GetByUsername("alice")
	if err != nil || user.Username != "alice" {
		t.Fatalf("expected success, got user=%v err=%v", user, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("none", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err = repo.GetByUsername("none")
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected not found, got %v", err)
	}
	assertMockExpectations(t, mock)
}

func TestGetByEmail_SuccessAndNotFound(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("bob@example.com", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "provider", "provider_id"}).AddRow(2, "bob", "bob@example.com", "", ""))

	user, err := repo.GetByEmail("bob@example.com")
	if err != nil || user.Email != "bob@example.com" {
		t.Fatalf("expected success, got user=%v err=%v", user, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("none@example.com", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err = repo.GetByEmail("none@example.com")
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected not found, got %v", err)
	}
	assertMockExpectations(t, mock)
}

func TestGetByProviderID_SuccessAndNotFound(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE provider = $1 AND provider_id = $2 ORDER BY "users"."id" LIMIT $3`)).
		WithArgs("google", "gid", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "provider", "provider_id"}).AddRow(3, "charlie", "charlie@example.com", "google", "gid"))

	user, err := repo.GetByProviderID("google", "gid")
	if err != nil || user.ProviderID != "gid" {
		t.Fatalf("expected success, got user=%v err=%v", user, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE provider = $1 AND provider_id = $2 ORDER BY "users"."id" LIMIT $3`)).
		WithArgs("google", "none", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err = repo.GetByProviderID("google", "none")
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected not found, got %v", err)
	}
	assertMockExpectations(t, mock)
}

func TestGetByID_SuccessAndNotFound(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(uint(9), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "provider", "provider_id"}).AddRow(9, "dave", "dave@example.com", "", ""))

	got, err := repo.GetByID(9)
	if err != nil || got.ID != 9 {
		t.Fatalf("expected success, got user=%v err=%v", got, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(uint(9999), 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err = repo.GetByID(9999)
	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected not found, got %v", err)
	}
	assertMockExpectations(t, mock)
}

func TestGetMethods_DBError(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("x", 1).
		WillReturnError(errors.New("db down"))

	if _, err := repo.GetByUsername("x"); err == nil || !strings.Contains(err.Error(), "failed to get user") {
		t.Fatalf("expected db error from GetByUsername, got %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs("x", 1).
		WillReturnError(errors.New("db down"))
	if _, err := repo.GetByEmail("x"); err == nil || !strings.Contains(err.Error(), "failed to get user") {
		t.Fatalf("expected db error from GetByEmail, got %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE provider = $1 AND provider_id = $2 ORDER BY "users"."id" LIMIT $3`)).
		WithArgs("google", "x", 1).
		WillReturnError(errors.New("db down"))
	if _, err := repo.GetByProviderID("google", "x"); err == nil || !strings.Contains(err.Error(), "failed to get user") {
		t.Fatalf("expected db error from GetByProviderID, got %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(uint(1), 1).
		WillReturnError(errors.New("db down"))
	if _, err := repo.GetByID(1); err == nil || !strings.Contains(err.Error(), "failed to get user") {
		t.Fatalf("expected db error from GetByID, got %v", err)
	}
	assertMockExpectations(t, mock)
}

func TestUpdate_SuccessAndDBError(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	user := &domain.User{ID: 7, Username: "eve", Email: "eve@example.com"}
	user.Username = "eve2"

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WithArgs("eve2", "eve@example.com", "", "", sqlmock.AnyArg(), sqlmock.AnyArg(), uint(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.Update(user); err != nil {
		t.Fatalf("expected update success, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WithArgs("eve2", "eve@example.com", "", "", sqlmock.AnyArg(), sqlmock.AnyArg(), uint(7)).
		WillReturnError(errors.New("update fail"))
	mock.ExpectRollback()

	if err := repo.Update(user); err == nil || !strings.Contains(err.Error(), "failed to update user") {
		t.Fatalf("expected update db error, got %v", err)
	}
	assertMockExpectations(t, mock)
}

func TestDelete_SuccessNotFoundAndDBError(t *testing.T) {
	repo, mock, cleanup := setupMockRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "users" WHERE "users"."id" = $1`)).
		WithArgs(uint(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.Delete(5); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "users" WHERE "users"."id" = $1`)).
		WithArgs(uint(9999)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := repo.Delete(9999); err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected not found on delete, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "users" WHERE "users"."id" = $1`)).
		WithArgs(uint(1)).
		WillReturnError(errors.New("delete fail"))
	mock.ExpectRollback()

	if err := repo.Delete(1); err == nil || !strings.Contains(err.Error(), "failed to delete user") {
		t.Fatalf("expected delete db error, got %v", err)
	}
	assertMockExpectations(t, mock)
}
