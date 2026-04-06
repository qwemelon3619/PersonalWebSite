package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/gin-gonic/gin"
)

type fakeHandler struct{}

func (f *fakeHandler) UploadBlogImageHandler(c *gin.Context) { c.Status(http.StatusOK) }
func (f *fakeHandler) DeleteBlogImageHandler(c *gin.Context) { c.Status(http.StatusOK) }

type fakeContainerClient struct {
	err error
}

func (f *fakeContainerClient) CreateContainer(ctx context.Context, containerName string, o *azblob.CreateContainerOptions) (azblob.CreateContainerResponse, error) {
	return azblob.CreateContainerResponse{}, f.err
}

func TestHandleError_Nil(t *testing.T) {
	handleError(nil)
}

func TestHandleError_ErrorExitsProcess(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		handleError(errors.New("boom"))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleError_ErrorExitsProcess")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected process to exit with error")
	}
}

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerRoutes(r, &fakeHandler{})

	uploadReq := httptest.NewRequest(http.MethodPost, "/blog-image", nil)
	uploadW := httptest.NewRecorder()
	r.ServeHTTP(uploadW, uploadReq)
	if uploadW.Code != http.StatusOK {
		t.Fatalf("expected POST /blog-image route, got %d", uploadW.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/blog-image", nil)
	deleteW := httptest.NewRecorder()
	r.ServeHTTP(deleteW, deleteReq)
	if deleteW.Code != http.StatusOK {
		t.Fatalf("expected DELETE /blog-image route, got %d", deleteW.Code)
	}
}

func TestEnsureContainerExists_Success(t *testing.T) {
	err := ensureContainerExists(&fakeContainerClient{}, "blogcontainer")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestEnsureContainerExists_AlreadyExists(t *testing.T) {
	err := ensureContainerExists(&fakeContainerClient{
		err: &azcore.ResponseError{ErrorCode: string(bloberror.ContainerAlreadyExists)},
	}, "blogcontainer")
	if err != nil {
		t.Fatalf("expected nil error for already exists, got %v", err)
	}
}

func TestEnsureContainerExists_OtherError(t *testing.T) {
	err := ensureContainerExists(&fakeContainerClient{err: errors.New("boom")}, "blogcontainer")
	if err == nil {
		t.Fatalf("expected error")
	}
}
