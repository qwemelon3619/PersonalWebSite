package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/model"
)

type stubPostService struct {
	createPostFn       func(req model.CreatePostRequest, authorID uint) (*domain.Post, error)
	getPostFn          func(id uint) (*domain.Post, error)
	getPostsByFilterFn func(filter model.PostFilter) ([]*domain.Post, error)
	updatePostFn       func(id uint, req model.UpdatePostRequest, authorID uint) (*domain.Post, error)
	deletePostFn       func(id, authorID uint) error
	listTagsFn         func() ([]*domain.Tag, error)
}

func (s *stubPostService) CreatePost(req model.CreatePostRequest, authorID uint) (*domain.Post, error) {
	return s.createPostFn(req, authorID)
}
func (s *stubPostService) GetPost(id uint) (*domain.Post, error) { return s.getPostFn(id) }
func (s *stubPostService) GetPostsByFilter(filter model.PostFilter) ([]*domain.Post, error) {
	return s.getPostsByFilterFn(filter)
}
func (s *stubPostService) UpdatePost(id uint, req model.UpdatePostRequest, authorID uint) (*domain.Post, error) {
	return s.updatePostFn(id, req, authorID)
}
func (s *stubPostService) DeletePost(id, authorID uint) error { return s.deletePostFn(id, authorID) }
func (s *stubPostService) ListTags() ([]*domain.Tag, error)   { return s.listTagsFn() }

func jsonReq(t *testing.T, method, path string, payload any) *http.Request {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestCreatePost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		body       any
		headerUser string
		svcErr     error
		want       int
	}{
		{"bad json", "{", "1", nil, http.StatusBadRequest},
		{"missing user", model.CreatePostRequest{Title: "t", Content: "c"}, "", nil, http.StatusUnauthorized},
		{"invalid user", model.CreatePostRequest{Title: "t", Content: "c"}, "x", nil, http.StatusUnauthorized},
		{"service error", model.CreatePostRequest{Title: "t", Content: "c"}, "1", errors.New("fail"), http.StatusInternalServerError},
		{"success", model.CreatePostRequest{Title: "t", Content: "c"}, "1", nil, http.StatusCreated},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubPostService{
				createPostFn: func(req model.CreatePostRequest, authorID uint) (*domain.Post, error) {
					if tc.svcErr != nil {
						return nil, tc.svcErr
					}
					return &domain.Post{ID: 1, Title: req.Title}, nil
				},
				getPostFn:          func(id uint) (*domain.Post, error) { return nil, nil },
				getPostsByFilterFn: func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
				updatePostFn:       func(id uint, req model.UpdatePostRequest, authorID uint) (*domain.Post, error) { return nil, nil },
				deletePostFn:       func(id, authorID uint) error { return nil },
				listTagsFn:         func() ([]*domain.Tag, error) { return nil, nil },
			}
			h := NewPostHandler(svc)
			r := gin.New()
			r.POST("/posts", h.CreatePost)

			var req *http.Request
			if tc.name == "bad json" {
				req = httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString("{"))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = jsonReq(t, http.MethodPost, "/posts", tc.body)
			}
			if tc.headerUser != "" {
				req.Header.Set("X-User-Id", tc.headerUser)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("want %d got %d body=%s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestGetPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubPostService{
		createPostFn: func(req model.CreatePostRequest, authorID uint) (*domain.Post, error) { return nil, nil },
		getPostFn: func(id uint) (*domain.Post, error) {
			if id == 1 {
				return &domain.Post{ID: 1, Title: "ok"}, nil
			}
			return nil, errors.New("not found")
		},
		getPostsByFilterFn: func(filter model.PostFilter) ([]*domain.Post, error) { return nil, nil },
		updatePostFn:       func(id uint, req model.UpdatePostRequest, authorID uint) (*domain.Post, error) { return nil, nil },
		deletePostFn:       func(id, authorID uint) error { return nil },
		listTagsFn:         func() ([]*domain.Tag, error) { return nil, nil },
	}
	h := NewPostHandler(svc)
	r := gin.New()
	r.GET("/posts/:id", h.GetPost)

	{
		req := httptest.NewRequest(http.MethodGet, "/posts/abc", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400 got %d", w.Code)
		}
	}
	{
		req := httptest.NewRequest(http.MethodGet, "/posts/2", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("want 404 got %d", w.Code)
		}
	}
	{
		req := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("want 200 got %d", w.Code)
		}
	}
}

func TestGetPostsAndTags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var seenFilter model.PostFilter
	svc := &stubPostService{
		createPostFn: func(req model.CreatePostRequest, authorID uint) (*domain.Post, error) { return nil, nil },
		getPostFn:    func(id uint) (*domain.Post, error) { return nil, nil },
		getPostsByFilterFn: func(filter model.PostFilter) ([]*domain.Post, error) {
			seenFilter = filter
			return []*domain.Post{{ID: 1}}, nil
		},
		updatePostFn: func(id uint, req model.UpdatePostRequest, authorID uint) (*domain.Post, error) { return nil, nil },
		deletePostFn: func(id, authorID uint) error { return nil },
		listTagsFn: func() ([]*domain.Tag, error) {
			return []*domain.Tag{{ID: 1, Name: "go"}}, nil
		},
	}
	h := NewPostHandler(svc)
	r := gin.New()
	r.GET("/posts", h.GetPosts)
	r.GET("/tags", h.GetTags)

	req := httptest.NewRequest(http.MethodGet, "/posts?author_id=9&published=true&limit=20&offset=10&search=abc&tag=go", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", w.Code)
	}
	if seenFilter.AuthorID == nil || *seenFilter.AuthorID != 9 {
		t.Fatalf("author filter not parsed")
	}
	if seenFilter.Published == nil || !*seenFilter.Published {
		t.Fatalf("published filter not parsed")
	}
	if seenFilter.Limit != 20 || seenFilter.Offset != 10 {
		t.Fatalf("limit/offset not parsed")
	}
	if seenFilter.Search == nil || *seenFilter.Search != "abc" || seenFilter.Tag == nil || *seenFilter.Tag != "go" {
		t.Fatalf("search/tag not parsed")
	}

	tagReq := httptest.NewRequest(http.MethodGet, "/tags", nil)
	tagW := httptest.NewRecorder()
	r.ServeHTTP(tagW, tagReq)
	if tagW.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", tagW.Code)
	}

	invalidReq := httptest.NewRequest(http.MethodGet, "/posts?author_id=x&published=zzz&limit=a&offset=b", nil)
	invalidW := httptest.NewRecorder()
	r.ServeHTTP(invalidW, invalidReq)
	if invalidW.Code != http.StatusOK {
		t.Fatalf("invalid query should be ignored and still return 200, got %d", invalidW.Code)
	}
}

func TestGetPostsAndTags_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubPostService{
		createPostFn: func(req model.CreatePostRequest, authorID uint) (*domain.Post, error) { return nil, nil },
		getPostFn:    func(id uint) (*domain.Post, error) { return nil, nil },
		getPostsByFilterFn: func(filter model.PostFilter) ([]*domain.Post, error) {
			return nil, errors.New("list fail")
		},
		updatePostFn: func(id uint, req model.UpdatePostRequest, authorID uint) (*domain.Post, error) { return nil, nil },
		deletePostFn: func(id, authorID uint) error { return nil },
		listTagsFn: func() ([]*domain.Tag, error) {
			return nil, errors.New("tags fail")
		},
	}
	h := NewPostHandler(svc)
	r := gin.New()
	r.GET("/posts", h.GetPosts)
	r.GET("/tags", h.GetTags)

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/posts", nil))
	if w1.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/tags", nil))
	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 got %d", w2.Code)
	}
}

func TestUpdateAndDeletePost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	title := "new"
	svc := &stubPostService{
		createPostFn: func(req model.CreatePostRequest, authorID uint) (*domain.Post, error) { return nil, nil },
		getPostFn:    func(id uint) (*domain.Post, error) { return nil, nil },
		getPostsByFilterFn: func(filter model.PostFilter) ([]*domain.Post, error) {
			return nil, nil
		},
		updatePostFn: func(id uint, req model.UpdatePostRequest, authorID uint) (*domain.Post, error) {
			if id == 2 {
				return nil, errors.New("forbidden")
			}
			return &domain.Post{ID: id, Title: *req.Title}, nil
		},
		deletePostFn: func(id, authorID uint) error {
			if id == 2 {
				return errors.New("forbidden")
			}
			return nil
		},
		listTagsFn: func() ([]*domain.Tag, error) { return nil, nil },
	}
	h := NewPostHandler(svc)
	r := gin.New()
	r.PUT("/posts/:id", h.UpdatePost)
	r.DELETE("/posts/:id", h.DeletePost)

	// update invalid id
	req := jsonReq(t, http.MethodPut, "/posts/x", model.UpdatePostRequest{Title: &title})
	req.Header.Set("X-User-Id", "1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", w.Code)
	}

	// update unauthorized header
	req2 := jsonReq(t, http.MethodPut, "/posts/1", model.UpdatePostRequest{Title: &title})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", w2.Code)
	}

	// update bad json
	req2c := httptest.NewRequest(http.MethodPut, "/posts/1", bytes.NewBufferString("{"))
	req2c.Header.Set("X-User-Id", "1")
	req2c.Header.Set("Content-Type", "application/json")
	w2c := httptest.NewRecorder()
	r.ServeHTTP(w2c, req2c)
	if w2c.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", w2c.Code)
	}

	req2b := jsonReq(t, http.MethodPut, "/posts/1", model.UpdatePostRequest{Title: &title})
	req2b.Header.Set("X-User-Id", "x")
	w2b := httptest.NewRecorder()
	r.ServeHTTP(w2b, req2b)
	if w2b.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", w2b.Code)
	}

	// update service error
	req3 := jsonReq(t, http.MethodPut, "/posts/2", model.UpdatePostRequest{Title: &title})
	req3.Header.Set("X-User-Id", "1")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", w3.Code)
	}

	// update success
	req4 := jsonReq(t, http.MethodPut, "/posts/1", model.UpdatePostRequest{Title: &title})
	req4.Header.Set("X-User-Id", "1")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("want 200 got %d", w4.Code)
	}

	// delete invalid id
	d1 := httptest.NewRequest(http.MethodDelete, "/posts/x", nil)
	d1.Header.Set("X-User-Id", "1")
	dw1 := httptest.NewRecorder()
	r.ServeHTTP(dw1, d1)
	if dw1.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", dw1.Code)
	}

	// delete unauthorized
	d2 := httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	dw2 := httptest.NewRecorder()
	r.ServeHTTP(dw2, d2)
	if dw2.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", dw2.Code)
	}

	d2b := httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	d2b.Header.Set("X-User-Id", "x")
	dw2b := httptest.NewRecorder()
	r.ServeHTTP(dw2b, d2b)
	if dw2b.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", dw2b.Code)
	}

	// delete service error
	d3 := httptest.NewRequest(http.MethodDelete, "/posts/2", nil)
	d3.Header.Set("X-User-Id", "1")
	dw3 := httptest.NewRecorder()
	r.ServeHTTP(dw3, d3)
	if dw3.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", dw3.Code)
	}

	// delete success
	d4 := httptest.NewRequest(http.MethodDelete, "/posts/1", nil)
	d4.Header.Set("X-User-Id", "1")
	dw4 := httptest.NewRecorder()
	r.ServeHTTP(dw4, d4)
	if dw4.Code != http.StatusNoContent {
		t.Fatalf("want 204 got %d", dw4.Code)
	}
}
