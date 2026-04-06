package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

type stubRepo struct {
	uploadFn func(ctx context.Context, file []byte, filePath string, contentType string) error
	deleteFn func(ctx context.Context, filePath string) error
}

func (s *stubRepo) UploadBlogImageToBlob(ctx context.Context, file []byte, filePath string, contentType string) error {
	return s.uploadFn(ctx, file, filePath, contentType)
}
func (s *stubRepo) DeleteBlob(ctx context.Context, filePath string) error {
	return s.deleteFn(ctx, filePath)
}

func TestUploadBlogImage_DataURIAndPath(t *testing.T) {
	origUUID := generateUUID
	generateUUID = func() string { return "fixed-uuid" }
	t.Cleanup(func() { generateUUID = origUUID })

	called := false
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error {
			called = true
			if string(file) != "hello" {
				t.Fatalf("expected decoded payload hello, got %q", string(file))
			}
			if filePath != "1/blog/img/fixed-uuid.png" {
				t.Fatalf("unexpected filePath %q", filePath)
			}
			if contentType != "image/png" {
				t.Fatalf("unexpected contentType %q", contentType)
			}
			return nil
		},
		deleteFn: func(ctx context.Context, filePath string) error { return nil },
	})

	resp, err := svc.UploadBlogImage(context.Background(), "a.png", "data:image/png;base64,"+base64.StdEncoding.EncodeToString([]byte("hello")), 1)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !called {
		t.Fatalf("expected upload repo to be called")
	}
	if resp.URL != "1/blog/img/fixed-uuid.png" || resp.Name != "a.png" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestUploadBlogImage_RawBase64(t *testing.T) {
	origUUID := generateUUID
	generateUUID = func() string { return "raw-uuid" }
	t.Cleanup(func() { generateUUID = origUUID })

	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
		deleteFn: func(ctx context.Context, filePath string) error { return nil },
	})

	_, err := svc.UploadBlogImage(context.Background(), "a.jpg", base64.StdEncoding.EncodeToString([]byte("hello")), 2)
	if err != nil {
		t.Fatalf("expected success for raw base64, got %v", err)
	}
}

func TestUploadBlogImage_InvalidBase64(t *testing.T) {
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
		deleteFn: func(ctx context.Context, filePath string) error { return nil },
	})
	_, err := svc.UploadBlogImage(context.Background(), "a.jpg", "!!not-base64!!", 1)
	if err == nil {
		t.Fatalf("expected base64 decode error")
	}
}

func TestUploadBlogImage_ContentTypeMapping(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{name: "jpg", file: "a.jpg", want: "image/jpeg"},
		{name: "jpeg", file: "a.jpeg", want: "image/jpeg"},
		{name: "gif", file: "a.gif", want: "image/gif"},
		{name: "webp", file: "a.webp", want: "image/webp"},
		{name: "default", file: "a.txt", want: "image/png"},
		{name: "noext", file: "a", want: "image/png"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origUUID := generateUUID
			generateUUID = func() string { return "map-uuid" }
			t.Cleanup(func() { generateUUID = origUUID })

			svc := NewImgService(&stubRepo{
				uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error {
					if contentType != tc.want {
						t.Fatalf("want %q got %q", tc.want, contentType)
					}
					return nil
				},
				deleteFn: func(ctx context.Context, filePath string) error { return nil },
			})
			_, err := svc.UploadBlogImage(context.Background(), tc.file, base64.StdEncoding.EncodeToString([]byte("x")), 1)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

func TestUploadBlogImage_RepoError(t *testing.T) {
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error {
			return errors.New("repo error")
		},
		deleteFn: func(ctx context.Context, filePath string) error { return nil },
	})
	_, err := svc.UploadBlogImage(context.Background(), "a.png", base64.StdEncoding.EncodeToString([]byte("x")), 1)
	if err == nil || !strings.Contains(err.Error(), "repo error") {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestUploadBlogImage_ResponseSizeUsesEncodedDataLength(t *testing.T) {
	origUUID := generateUUID
	generateUUID = func() string { return "size-uuid" }
	t.Cleanup(func() { generateUUID = origUUID })

	raw := "hello"
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
		deleteFn: func(ctx context.Context, filePath string) error { return nil },
	})
	resp, err := svc.UploadBlogImage(context.Background(), "a.png", encoded, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Size != int64(len(encoded)) {
		t.Fatalf("expected size=%d got %d", len(encoded), resp.Size)
	}
}

func TestDeleteBlogImage_RepoCalled(t *testing.T) {
	called := false
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
		deleteFn: func(ctx context.Context, filePath string) error {
			called = true
			if filePath != "1/blog/img/a.png" {
				t.Fatalf("unexpected filePath %q", filePath)
			}
			return nil
		},
	})
	if err := svc.DeleteBlogImage(context.Background(), "1/blog/img/a.png"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !called {
		t.Fatalf("expected delete to be called")
	}
}

func TestDeleteBlogImage_RepoError(t *testing.T) {
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
		deleteFn: func(ctx context.Context, filePath string) error { return errors.New("delete fail") },
	})
	if err := svc.DeleteBlogImage(context.Background(), "x"); err == nil {
		t.Fatalf("expected error")
	}
}
