package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/model"
)

type stubBlogImageService struct {
	uploadFn func(ctx context.Context, filename string, data string, userId int) (*model.ImageResponse, error)
	deleteFn func(ctx context.Context, filePath string) error
}

func (s *stubBlogImageService) UploadBlogImage(ctx context.Context, filename string, data string, userId int) (*model.ImageResponse, error) {
	return s.uploadFn(ctx, filename, data, userId)
}
func (s *stubBlogImageService) DeleteBlogImage(ctx context.Context, filePath string) error {
	return s.deleteFn(ctx, filePath)
}

func TestUploadBlogImageHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &imageHandler{service: &stubBlogImageService{}}
	r := gin.New()
	r.POST("/blog-image", h.UploadBlogImageHandler)

	req := httptest.NewRequest(http.MethodPost, "/blog-image", strings.NewReader(`{"filename"`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUploadBlogImageHandler_InvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &imageHandler{service: &stubBlogImageService{}}
	r := gin.New()
	r.POST("/blog-image", h.UploadBlogImageHandler)

	req := httptest.NewRequest(http.MethodPost, "/blog-image", strings.NewReader(`{"filename":"a.png","userId":"abc","data":"dGVzdA=="}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUploadBlogImageHandler_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &imageHandler{service: &stubBlogImageService{
		uploadFn: func(ctx context.Context, filename string, data string, userId int) (*model.ImageResponse, error) {
			return nil, errors.New("upload failed")
		},
	}}
	r := gin.New()
	r.POST("/blog-image", h.UploadBlogImageHandler)

	req := httptest.NewRequest(http.MethodPost, "/blog-image", strings.NewReader(`{"filename":"a.png","userId":"1","data":"dGVzdA=="}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestUploadBlogImageHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &imageHandler{service: &stubBlogImageService{
		uploadFn: func(ctx context.Context, filename string, data string, userId int) (*model.ImageResponse, error) {
			return &model.ImageResponse{URL: "1/blog/img/x.png", Name: "a.png", Size: 12}, nil
		},
	}}
	r := gin.New()
	r.POST("/blog-image", h.UploadBlogImageHandler)

	req := httptest.NewRequest(http.MethodPost, "/blog-image", strings.NewReader(`{"filename":"a.png","userId":"1","data":"dGVzdA=="}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDeleteBlogImageHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &imageHandler{service: &stubBlogImageService{}}
	r := gin.New()
	r.DELETE("/blog-image", h.DeleteBlogImageHandler)

	req := httptest.NewRequest(http.MethodDelete, "/blog-image", strings.NewReader(`{"path"`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteBlogImageHandler_EmptyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &imageHandler{service: &stubBlogImageService{}}
	r := gin.New()
	r.DELETE("/blog-image", h.DeleteBlogImageHandler)

	req := httptest.NewRequest(http.MethodDelete, "/blog-image", strings.NewReader(`{"path":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDeleteBlogImageHandler_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &imageHandler{service: &stubBlogImageService{
		deleteFn: func(ctx context.Context, filePath string) error { return errors.New("delete failed") },
	}}
	r := gin.New()
	r.DELETE("/blog-image", h.DeleteBlogImageHandler)

	req := httptest.NewRequest(http.MethodDelete, "/blog-image", strings.NewReader(`{"path":"1/blog/img/x.png"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestDeleteBlogImageHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &imageHandler{service: &stubBlogImageService{
		deleteFn: func(ctx context.Context, filePath string) error { return nil },
	}}
	r := gin.New()
	r.DELETE("/blog-image", h.DeleteBlogImageHandler)

	req := httptest.NewRequest(http.MethodDelete, "/blog-image", strings.NewReader(`{"path":"1/blog/img/x.png"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
