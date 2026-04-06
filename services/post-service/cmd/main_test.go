package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/pkg/logger"
)

type fakePostHandler struct{}

func (f *fakePostHandler) GetPosts(c *gin.Context)   { c.Status(http.StatusOK) }
func (f *fakePostHandler) GetPost(c *gin.Context)    { c.Status(http.StatusOK) }
func (f *fakePostHandler) GetTags(c *gin.Context)    { c.Status(http.StatusOK) }
func (f *fakePostHandler) CreatePost(c *gin.Context) { c.Status(http.StatusCreated) }
func (f *fakePostHandler) UpdatePost(c *gin.Context) { c.Status(http.StatusOK) }
func (f *fakePostHandler) DeletePost(c *gin.Context) { c.Status(http.StatusNoContent) }

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerRoutes(r, &fakePostHandler{}, logger.New("test"))

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/health", http.StatusOK},
		{http.MethodGet, "/posts", http.StatusOK},
		{http.MethodGet, "/posts/1", http.StatusOK},
		{http.MethodGet, "/tags", http.StatusOK},
		{http.MethodPost, "/posts", http.StatusCreated},
		{http.MethodPut, "/posts/1", http.StatusOK},
		{http.MethodDelete, "/posts/1", http.StatusNoContent},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Fatalf("%s %s want %d got %d", tc.method, tc.path, tc.want, w.Code)
		}
	}
}
