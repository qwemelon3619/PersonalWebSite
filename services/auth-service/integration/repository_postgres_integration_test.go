//go:build integration

package integration

import (
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/auth-service/internal/repository"
)

func TestUserRepository_PostgresIntegration(t *testing.T) {
	dsn := os.Getenv("AUTH_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("AUTH_TEST_POSTGRES_DSN is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect postgres: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := repository.NewUserRepository(db)

	user := &domain.User{
		Username:   "integration-user",
		Email:      "integration-user@example.com",
		Provider:   "google",
		ProviderID: "integration-provider-id",
	}
	if err := repo.Create(user); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	gotByEmail, err := repo.GetByEmail(user.Email)
	if err != nil || gotByEmail.ID != user.ID {
		t.Fatalf("get by email failed: user=%v err=%v", gotByEmail, err)
	}

	gotByProvider, err := repo.GetByProviderID("google", "integration-provider-id")
	if err != nil || gotByProvider.ID != user.ID {
		t.Fatalf("get by provider failed: user=%v err=%v", gotByProvider, err)
	}

	user.Username = "integration-user-updated"
	if err := repo.Update(user); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	gotByID, err := repo.GetByID(user.ID)
	if err != nil || gotByID.Username != "integration-user-updated" {
		t.Fatalf("get by id after update failed: user=%v err=%v", gotByID, err)
	}

	if err := repo.Delete(user.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if err := repo.Delete(user.ID); err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Fatalf("expected user not found after delete, got %v", err)
	}
}
