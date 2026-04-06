//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/handler"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/model"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/repository"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/service"
)

func setupIntegration(t *testing.T) (*azblob.Client, config.BlobConfig, *gin.Engine) {
	t.Helper()

	conn := os.Getenv("AUTH_TEST_AZURE_STORAGE_CONNECTION_STRING")
	container := os.Getenv("AUTH_TEST_BLOB_CONTAINER")
	if conn == "" || container == "" {
		t.Skip("AUTH_TEST_AZURE_STORAGE_CONNECTION_STRING / AUTH_TEST_BLOB_CONTAINER not set")
	}

	client, err := azblob.NewClientFromConnectionString(conn, nil)
	if err != nil {
		t.Fatalf("failed to create azblob client: %v", err)
	}

	_, err = client.CreateContainer(context.Background(), container, &azblob.CreateContainerOptions{
		Access: to.Ptr(azblob.PublicAccessTypeBlob),
	})
	if err != nil {
		var respErr *azcore.ResponseError
		if !(errors.As(err, &respErr) && respErr.ErrorCode == string(bloberror.ContainerAlreadyExists)) {
			t.Fatalf("failed to create container: %v", err)
		}
	}

	cfg := config.BlobConfig{BlobContainerName: container}
	repo := repository.NewImgRepository(client, cfg)
	svc := service.NewImgService(repo)
	h := handler.NewBlogImageHandler(svc)

	r := gin.New()
	r.POST("/blog-image", h.UploadBlogImageHandler)
	r.DELETE("/blog-image", h.DeleteBlogImageHandler)

	return client, cfg, r
}

func TestAzurite_UploadAndDeleteBlobE2E(t *testing.T) {
	client, cfg, r := setupIntegration(t)
	ctx := context.Background()

	uploadPayload := model.UploadImageRequest{
		Filename: "integration.png",
		UserId:   "77",
		Data:     "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("hello-e2e")),
	}
	b, _ := json.Marshal(uploadPayload)
	upReq := httptest.NewRequest(http.MethodPost, "/blog-image", bytes.NewReader(b))
	upReq.Header.Set("Content-Type", "application/json")
	upW := httptest.NewRecorder()
	r.ServeHTTP(upW, upReq)
	if upW.Code != http.StatusOK {
		t.Fatalf("upload failed status=%d body=%s", upW.Code, upW.Body.String())
	}

	var upResp model.ImageResponse
	if err := json.Unmarshal(upW.Body.Bytes(), &upResp); err != nil {
		t.Fatalf("failed to parse upload response: %v", err)
	}
	if upResp.URL == "" {
		t.Fatalf("expected blob path in response")
	}

	_, err := client.DownloadStream(ctx, cfg.BlobContainerName, upResp.URL, nil)
	if err != nil {
		t.Fatalf("expected uploaded blob to exist, got %v", err)
	}

	delPayload := model.DeleteImageRequest{Path: upResp.URL}
	db, _ := json.Marshal(delPayload)
	delReq := httptest.NewRequest(http.MethodDelete, "/blog-image", bytes.NewReader(db))
	delReq.Header.Set("Content-Type", "application/json")
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("delete failed status=%d body=%s", delW.Code, delW.Body.String())
	}

	_, err = client.DownloadStream(ctx, cfg.BlobContainerName, upResp.URL, nil)
	if err == nil {
		t.Fatalf("expected blob to be deleted")
	}
}

func TestAPI_InvalidPayloads(t *testing.T) {
	_, _, r := setupIntegration(t)

	// invalid upload payload -> 400
	upReq := httptest.NewRequest(http.MethodPost, "/blog-image", bytes.NewReader([]byte(`{"filename":"a.png"}`)))
	upReq.Header.Set("Content-Type", "application/json")
	upW := httptest.NewRecorder()
	r.ServeHTTP(upW, upReq)
	if upW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid upload payload, got %d", upW.Code)
	}

	// bad base64 -> 500
	upReq2 := httptest.NewRequest(http.MethodPost, "/blog-image", bytes.NewReader([]byte(`{"filename":"a.png","userId":"1","data":"!!!bad!!!"}`)))
	upReq2.Header.Set("Content-Type", "application/json")
	upW2 := httptest.NewRecorder()
	r.ServeHTTP(upW2, upReq2)
	if upW2.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for bad base64, got %d", upW2.Code)
	}
}
