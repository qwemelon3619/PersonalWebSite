package repository

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/model"
)

func setupMockPostRepo(t *testing.T) (*postRepository, sqlmock.Sqlmock, func()) {
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
	return &postRepository{db: gdb}, mock, cleanup
}

func assertPostMock(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPostRepository_Create(t *testing.T) {
	repo, mock, cleanup := setupMockPostRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "posts"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()
	p1 := &domain.Post{Title: "t1", Content: "c1", AuthorID: 1, Published: true}
	if err := repo.Create(p1); err != nil {
		t.Fatalf("create published: %v", err)
	}
	if p1.PublishedAt == nil {
		t.Fatalf("expected published_at set")
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "posts"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	mock.ExpectCommit()
	p2 := &domain.Post{Title: "t2", Content: "c2", AuthorID: 1, Published: false}
	if err := repo.Create(p2); err != nil {
		t.Fatalf("create unpublished: %v", err)
	}
	if p2.PublishedAt != nil {
		t.Fatalf("expected published_at nil")
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "posts"`)).
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()
	if err := repo.Create(&domain.Post{Title: "t3", Content: "c3", AuthorID: 1}); err == nil || !strings.Contains(err.Error(), "failed to create post") {
		t.Fatalf("expected create db error, got %v", err)
	}

	assertPostMock(t, mock)
}

func TestPostRepository_GetByID(t *testing.T) {
	repo, mock, cleanup := setupMockPostRepo(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(1), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "author_id", "published"}).AddRow(1, "t1", "c1", 10, true))
	mock.ExpectQuery(`SELECT \* FROM "post_tags" WHERE "post_tags"\."post_id" = \$1`).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"post_id", "tag_id"}).AddRow(1, 3))
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
		WithArgs(uint(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email"}).AddRow(10, "u1", "u1@example.com"))
	mock.ExpectQuery(`SELECT \* FROM "tags" WHERE "tags"\."id" = \$1`).
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(3, "go"))

	got, err := repo.GetByID(1)
	if err != nil {
		t.Fatalf("get by id success: %v", err)
	}
	if got.Author.ID != 10 || len(got.Tags) != 1 {
		t.Fatalf("expected author/tags preload, got author=%d tags=%d", got.Author.ID, len(got.Tags))
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(999), 1).
		WillReturnError(gorm.ErrRecordNotFound)
	if _, err := repo.GetByID(999); err == nil || !strings.Contains(err.Error(), "post not found") {
		t.Fatalf("expected not found, got %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "posts" WHERE "posts"."id" = $1 ORDER BY "posts"."id" LIMIT $2`)).
		WithArgs(uint(2), 1).
		WillReturnError(errors.New("db down"))
	if _, err := repo.GetByID(2); err == nil || !strings.Contains(err.Error(), "failed to get post") {
		t.Fatalf("expected db error, got %v", err)
	}

	assertPostMock(t, mock)
}

func TestPostRepository_GetAll(t *testing.T) {
	repo, mock, cleanup := setupMockPostRepo(t)
	defer cleanup()

	emptyRows := sqlmock.NewRows([]string{"id", "title", "content", "author_id", "published"})

	authorID := uint(7)
	mock.ExpectQuery(`SELECT \* FROM "posts" WHERE author_id = \$1 ORDER BY created_at DESC`).
		WithArgs(authorID).
		WillReturnRows(emptyRows)
	if _, err := repo.GetAll(model.PostFilter{AuthorID: &authorID}); err != nil {
		t.Fatalf("author filter: %v", err)
	}

	published := true
	mock.ExpectQuery(`SELECT \* FROM "posts" WHERE published = \$1 ORDER BY created_at DESC`).
		WithArgs(published).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "author_id", "published"}))
	if _, err := repo.GetAll(model.PostFilter{Published: &published}); err != nil {
		t.Fatalf("published filter: %v", err)
	}

	search := "go"
	mock.ExpectQuery(`SELECT \* FROM "posts" WHERE title ILIKE \$1 OR content ILIKE \$2 ORDER BY created_at DESC`).
		WithArgs("%go%", "%go%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "author_id", "published"}))
	if _, err := repo.GetAll(model.PostFilter{Search: &search}); err != nil {
		t.Fatalf("search filter: %v", err)
	}

	tag := "go"
	mock.ExpectQuery(`SELECT .* FROM "posts" JOIN post_tags pt ON pt\.post_id = posts\.id JOIN tags t ON t\.id = pt\.tag_id WHERE t\.name = \$1 ORDER BY posts\.created_at DESC`).
		WithArgs(tag).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "author_id", "published"}))
	if _, err := repo.GetAll(model.PostFilter{Tag: &tag, OrderBy: "posts.created_at DESC"}); err != nil {
		t.Fatalf("tag filter: %v", err)
	}

	mock.ExpectQuery(`SELECT \* FROM "posts" ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "author_id", "published"}))
	if _, err := repo.GetAll(model.PostFilter{Limit: 1, Offset: 1}); err != nil {
		t.Fatalf("limit/offset: %v", err)
	}

	mock.ExpectQuery(`SELECT \* FROM "posts" ORDER BY created_at DESC`).
		WillReturnError(errors.New("list fail"))
	if _, err := repo.GetAll(model.PostFilter{}); err == nil || !strings.Contains(err.Error(), "failed to list posts") {
		t.Fatalf("expected get all db error, got %v", err)
	}

	assertPostMock(t, mock)
}

func TestPostRepository_Update(t *testing.T) {
	repo, mock, cleanup := setupMockPostRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "posts" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	p := &domain.Post{ID: 1, Title: "t", Content: "c", Published: true}
	if err := repo.Update(p); err != nil {
		t.Fatalf("update success: %v", err)
	}
	if p.PublishedAt == nil {
		t.Fatalf("expected published_at set")
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "posts" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	p.Published = false
	if err := repo.Update(p); err != nil {
		t.Fatalf("update unpublish: %v", err)
	}
	if p.PublishedAt != nil {
		t.Fatalf("expected published_at nil")
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "posts" SET`)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	if err := repo.Update(&domain.Post{ID: 999, Title: "x", Content: "y"}); err == nil || !strings.Contains(err.Error(), "post not found") {
		t.Fatalf("expected not found, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "posts" SET`)).
		WillReturnError(errors.New("update fail"))
	mock.ExpectRollback()
	if err := repo.Update(&domain.Post{ID: 2, Title: "x", Content: "y"}); err == nil || !strings.Contains(err.Error(), "failed to update post") {
		t.Fatalf("expected update db error, got %v", err)
	}

	assertPostMock(t, mock)
}

func TestPostRepository_DeleteAndGetByAuthorID(t *testing.T) {
	repo, mock, cleanup := setupMockPostRepo(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "posts" WHERE "posts"."id" = $1`)).
		WithArgs(uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Delete(1); err != nil {
		t.Fatalf("delete success: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "posts" WHERE "posts"."id" = $1`)).
		WithArgs(uint(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	if err := repo.Delete(999); err == nil || !strings.Contains(err.Error(), "post not found") {
		t.Fatalf("expected delete not found, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "posts" WHERE "posts"."id" = $1`)).
		WithArgs(uint(2)).
		WillReturnError(errors.New("delete fail"))
	mock.ExpectRollback()
	if err := repo.Delete(2); err == nil || !strings.Contains(err.Error(), "failed to delete post") {
		t.Fatalf("expected delete db error, got %v", err)
	}

	authorID := uint(5)
	mock.ExpectQuery(`SELECT \* FROM "posts" WHERE author_id = \$1 ORDER BY created_at DESC`).
		WithArgs(authorID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "content", "author_id", "published"}))
	if _, err := repo.GetByAuthorID(authorID); err != nil {
		t.Fatalf("get by author: %v", err)
	}

	assertPostMock(t, mock)
}
