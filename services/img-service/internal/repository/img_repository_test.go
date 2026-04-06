package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/config"
)

type mockBlobClient struct {
	uploadCalled bool
	deleteCalled bool

	uploadContainer string
	uploadPath      string
	uploadData      []byte
	uploadOptions   *azblob.UploadBufferOptions
	uploadErr       error

	deleteContainer string
	deletePath      string
	deleteErr       error
}

func (m *mockBlobClient) UploadBuffer(ctx context.Context, containerName string, blobName string, data []byte, o *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error) {
	m.uploadCalled = true
	m.uploadContainer = containerName
	m.uploadPath = blobName
	m.uploadData = data
	m.uploadOptions = o
	return azblob.UploadBufferResponse{}, m.uploadErr
}

func (m *mockBlobClient) DeleteBlob(ctx context.Context, containerName string, blobName string, o *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error) {
	m.deleteCalled = true
	m.deleteContainer = containerName
	m.deletePath = blobName
	return azblob.DeleteBlobResponse{}, m.deleteErr
}

func testConfig() config.BlobConfig {
	return config.BlobConfig{BlobContainerName: "blogcontainer"}
}

func TestUploadBlogImageToBlob_NilClient(t *testing.T) {
	repo := NewImgRepository(nil, testConfig())
	err := repo.UploadBlogImageToBlob(context.Background(), []byte("x"), "1/blog/img/a.png", "image/png")
	if err == nil || err.Error() != "Azure container client is nil" {
		t.Fatalf("expected nil client error, got %v", err)
	}
}

func TestUploadBlogImageToBlob_UploadCalledWithArgs(t *testing.T) {
	mock := &mockBlobClient{}
	repo := NewImgRepository(mock, testConfig())
	err := repo.UploadBlogImageToBlob(context.Background(), []byte("hello"), "1/blog/img/a.png", "image/png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.uploadCalled {
		t.Fatalf("expected upload call")
	}
	if mock.uploadContainer != "blogcontainer" || mock.uploadPath != "1/blog/img/a.png" {
		t.Fatalf("unexpected args container=%q path=%q", mock.uploadContainer, mock.uploadPath)
	}
	if string(mock.uploadData) != "hello" {
		t.Fatalf("unexpected upload data: %q", string(mock.uploadData))
	}
	if mock.uploadOptions == nil || mock.uploadOptions.HTTPHeaders == nil || mock.uploadOptions.HTTPHeaders.BlobContentType == nil {
		t.Fatalf("expected blob content type header")
	}
	if got := *mock.uploadOptions.HTTPHeaders.BlobContentType; got != "image/png" {
		t.Fatalf("expected content type image/png, got %q", got)
	}
}

func TestUploadBlogImageToBlob_AzureError(t *testing.T) {
	mock := &mockBlobClient{uploadErr: errors.New("azure upload fail")}
	repo := NewImgRepository(mock, testConfig())
	err := repo.UploadBlogImageToBlob(context.Background(), []byte("x"), "1/blog/img/a.png", "image/png")
	if err == nil || err.Error() != "azure upload fail" {
		t.Fatalf("expected azure upload error, got %v", err)
	}
}

func TestDeleteBlob_NilClient(t *testing.T) {
	repo := NewImgRepository(nil, testConfig())
	err := repo.DeleteBlob(context.Background(), "1/blog/img/a.png")
	if err == nil || err.Error() != "Azure container client is nil" {
		t.Fatalf("expected nil client error, got %v", err)
	}
}

func TestDeleteBlob_DeleteCalledWithArgs(t *testing.T) {
	mock := &mockBlobClient{}
	repo := NewImgRepository(mock, testConfig())
	err := repo.DeleteBlob(context.Background(), "1/blog/img/a.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.deleteCalled {
		t.Fatalf("expected delete call")
	}
	if mock.deleteContainer != "blogcontainer" || mock.deletePath != "1/blog/img/a.png" {
		t.Fatalf("unexpected delete args container=%q path=%q", mock.deleteContainer, mock.deletePath)
	}
}

func TestDeleteBlob_AzureError(t *testing.T) {
	mock := &mockBlobClient{deleteErr: errors.New("azure delete fail")}
	repo := NewImgRepository(mock, testConfig())
	err := repo.DeleteBlob(context.Background(), "1/blog/img/a.png")
	if err == nil || err.Error() != "azure delete fail" {
		t.Fatalf("expected azure delete error, got %v", err)
	}
}
