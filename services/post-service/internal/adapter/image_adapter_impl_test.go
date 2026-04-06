package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
)

func TestUploadImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blog-image" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"URL": "/1/blog/img/u.jpg"})
	}))
	defer server.Close()

	a := NewImageAdapter(&config.PostConfig{ImageServiceURL: server.URL})
	url, err := a.UploadImage("data:image/png;base64,AAAA", 1)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if url != "/1/blog/img/u.jpg" {
		t.Fatalf("unexpected url: %q", url)
	}
}

func TestUploadImage_ErrorCases(t *testing.T) {
	// non-200
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer s1.Close()
	a1 := NewImageAdapter(&config.PostConfig{ImageServiceURL: s1.URL})
	if _, err := a1.UploadImage("x", 1); err == nil {
		t.Fatalf("expected status error")
	}

	// bad JSON
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer s2.Close()
	a2 := NewImageAdapter(&config.PostConfig{ImageServiceURL: s2.URL})
	if _, err := a2.UploadImage("x", 1); err == nil {
		t.Fatalf("expected decode error")
	}

	// network error
	a3 := NewImageAdapter(&config.PostConfig{ImageServiceURL: "http://127.0.0.1:1"})
	if _, err := a3.UploadImage("x", 1); err == nil {
		t.Fatalf("expected network error")
	}
}

func TestDeleteImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blog-image" || r.Method != http.MethodDelete {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	a := NewImageAdapter(&config.PostConfig{ImageServiceURL: server.URL})
	if err := a.DeleteImage("/1/blog/img/a.jpg"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestDeleteImage_ErrorCases(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer s1.Close()
	a1 := NewImageAdapter(&config.PostConfig{ImageServiceURL: s1.URL})
	if err := a1.DeleteImage("/x"); err == nil {
		t.Fatalf("expected status error")
	}

	a2 := NewImageAdapter(&config.PostConfig{ImageServiceURL: "http://127.0.0.1:1"})
	if err := a2.DeleteImage("/x"); err == nil {
		t.Fatalf("expected network error")
	}
}

func TestProcessMarkdownForImages(t *testing.T) {
	called := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		_ = json.NewEncoder(w).Encode(map[string]string{"URL": fmt.Sprintf("/1/blog/img/%d.jpg", called)})
	}))
	defer server.Close()
	a := NewImageAdapter(&config.PostConfig{ImageServiceURL: server.URL})

	in := "text ![a](data:image/png;base64,AAAA) and ![b](data:image/jpeg;base64,BBBB)"
	out, err := a.ProcessMarkdownForImages(in, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out, "![a](/1/blog/img/1.jpg)") || !strings.Contains(out, "![b](/1/blog/img/2.jpg)") {
		t.Fatalf("expected replaced urls, got %q", out)
	}
}

func TestProcessMarkdownForImages_KeepOriginalOnUploadError(t *testing.T) {
	a := NewImageAdapter(&config.PostConfig{ImageServiceURL: "http://127.0.0.1:1"})
	in := "![a](data:image/png;base64,AAAA)"
	out, err := a.ProcessMarkdownForImages(in, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out != in {
		t.Fatalf("expected original markdown on upload error, got %q", out)
	}
}

func TestExtractImageURLsFromContent(t *testing.T) {
	a := NewImageAdapter(&config.PostConfig{})
	in := "![a](/x.png) ![b](http://example.com/b.jpg) ![c](data:image/png;base64,AAA)"
	urls := a.ExtractImageURLsFromContent(in)
	if len(urls) != 2 {
		t.Fatalf("expected 2 urls, got %d: %v", len(urls), urls)
	}
}
