package repository

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockTagRepo(t *testing.T) (*tagRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	mock.MatchExpectationsInOrder(false)

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	cleanup := func() { _ = sqlDB.Close() }
	return &tagRepository{db: gdb}, mock, cleanup
}

func assertTagMock(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTagRepository_AttachTagsToPost(t *testing.T) {
	repo, mock, cleanup := setupMockTagRepo(t)
	defer cleanup()

	if err := repo.AttachTagsToPost(1, []string{}); err != nil {
		t.Fatalf("empty tags should be no-op: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(999), 1).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectRollback()
	if err := repo.AttachTagsToPost(999, []string{"go"}); err == nil || !strings.Contains(err.Error(), "post not found") {
		t.Fatalf("expected post not found, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "tags" WHERE name = \$1 .*ORDER BY "tags"\."id" LIMIT \$3`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tags"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectQuery(`INSERT INTO "tags" .*ON CONFLICT DO NOTHING RETURNING "id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
	mock.ExpectExec(`INSERT INTO "post_tags"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "posts" SET "updated_at"=\$1 WHERE "id" = \$2`).
		WithArgs(sqlmock.AnyArg(), uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.AttachTagsToPost(1, []string{"go"}); err != nil {
		t.Fatalf("attach success: %v", err)
	}

	assertTagMock(t, mock)
}

func TestTagRepository_ReplaceTagsForPost(t *testing.T) {
	repo, mock, cleanup := setupMockTagRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(999), 1).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectRollback()
	if err := repo.ReplaceTagsForPost(999, []string{"go"}); err == nil || !strings.Contains(err.Error(), "post not found") {
		t.Fatalf("expected post not found, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(`DELETE FROM "post_tags" WHERE "post_tags"\."post_id" = \$1`).
		WithArgs(uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "posts" SET "updated_at"=\$1 WHERE "id" = \$2`).
		WithArgs(sqlmock.AnyArg(), uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.ReplaceTagsForPost(1, []string{}); err != nil {
		t.Fatalf("replace empty tags: %v", err)
	}

	assertTagMock(t, mock)
}

func TestTagRepository_ReadAndDeleteMethods(t *testing.T) {
	repo, mock, cleanup := setupMockTagRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "post_tags" WHERE "post_tags"\."post_id" = \$1`).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"post_id", "tag_id"}).AddRow(1, 3))
	mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"\."id" = \$1`).
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(3, "go"))
	tags, err := repo.GetTagsForPost(1)
	if err != nil || len(tags) != 1 {
		t.Fatalf("get tags success failed: tags=%v err=%v", tags, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(2), 1).
		WillReturnError(errors.New("db fail"))
	if _, err := repo.GetTagsForPost(2); err == nil || !strings.Contains(err.Error(), "failed to load post tags") {
		t.Fatalf("expected get tags db error, got %v", err)
	}

	mock.ExpectQuery(`SELECT \* FROM "tags" ORDER BY name ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "go").AddRow(2, "web"))
	all, err := repo.ListTags()
	if err != nil || len(all) != 2 {
		t.Fatalf("list tags failed: tags=%v err=%v", all, err)
	}

	mock.ExpectQuery(`SELECT \* FROM "tags" ORDER BY name ASC`).
		WillReturnError(errors.New("list fail"))
	if _, err := repo.ListTags(); err == nil || !strings.Contains(err.Error(), "failed to list tags") {
		t.Fatalf("expected list db error, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "tags" WHERE "tags"."id" = $1`)).
		WithArgs(uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.DeleteTag(1); err != nil {
		t.Fatalf("delete tag success: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "tags" WHERE "tags"."id" = $1`)).
		WithArgs(uint(2)).
		WillReturnError(errors.New("delete fail"))
	mock.ExpectRollback()
	if err := repo.DeleteTag(2); err == nil || !strings.Contains(err.Error(), "failed to delete tag") {
		t.Fatalf("expected delete tag db error, got %v", err)
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "posts" JOIN post_tags ON posts\.id = post_tags\.post_id WHERE post_tags\.tag_id = \$1`).
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	if err := repo.DeleteUnusedTag(3); err != nil {
		t.Fatalf("delete unused (in use) should pass: %v", err)
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "posts" JOIN post_tags ON posts\.id = post_tags\.post_id WHERE post_tags\.tag_id = \$1`).
		WithArgs(uint(4)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "tags" WHERE "tags"."id" = $1`)).
		WithArgs(uint(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.DeleteUnusedTag(4); err != nil {
		t.Fatalf("delete unused should delete: %v", err)
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM "posts" JOIN post_tags ON posts\.id = post_tags\.post_id WHERE post_tags\.tag_id = \$1`).
		WithArgs(uint(5)).
		WillReturnError(errors.New("count fail"))
	if err := repo.DeleteUnusedTag(5); err == nil || !strings.Contains(err.Error(), "failed to count tag usage") {
		t.Fatalf("expected count db error, got %v", err)
	}

	assertTagMock(t, mock)
}
