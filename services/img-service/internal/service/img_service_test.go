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

func pngBytes() []byte {
	return append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("payload")...)
}

func jpegBytes() []byte {
	return append([]byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, []byte("payload")...)
}

func gifBytes() []byte { return append([]byte("GIF89a"), []byte("payload")...) }

func webpBytes() []byte {
	return append(append([]byte("RIFF"), []byte{0x00, 0x00, 0x00, 0x00}...), []byte("WEBPVP8 payload")...)
}

func encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func TestUploadBlogImage_DataURIAndPath(t *testing.T) {
	origUUID := generateUUID
	generateUUID = func() string { return "fixed-uuid" }
	t.Cleanup(func() { generateUUID = origUUID })

	raw := pngBytes()
	called := false
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error {
			called = true
			if string(file) != string(raw) {
				t.Fatalf("expected decoded payload, got %q", string(file))
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

	resp, err := svc.UploadBlogImage(context.Background(), "a.png", "data:image/png;base64,"+encode(raw), 1)
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

	_, err := svc.UploadBlogImage(context.Background(), "a.jpg", encode(jpegBytes()), 2)
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
		data []byte
		want string
	}{
		{name: "jpg", file: "a.jpg", data: jpegBytes(), want: "image/jpeg"},
		{name: "jpeg", file: "a.jpeg", data: jpegBytes(), want: "image/jpeg"},
		{name: "png", file: "a.png", data: pngBytes(), want: "image/png"},
		{name: "gif", file: "a.gif", data: gifBytes(), want: "image/gif"},
		{name: "webp", file: "a.webp", data: webpBytes(), want: "image/webp"},
		{name: "uppercase extension", file: "a.PNG", data: pngBytes(), want: "image/png"},
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
			_, err := svc.UploadBlogImage(context.Background(), tc.file, encode(tc.data), 1)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		})
	}
}

func TestUploadBlogImage_RejectsUnsupportedExtensions(t *testing.T) {
	for _, filename := range []string{"a.txt", "a", "a.svg", "a.html", "a.exe", "a.png.html"} {
		t.Run(filename, func(t *testing.T) {
			called := false
			svc := NewImgService(&stubRepo{
				uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error {
					called = true
					return nil
				},
				deleteFn: func(ctx context.Context, filePath string) error { return nil },
			})
			_, err := svc.UploadBlogImage(context.Background(), filename, encode(pngBytes()), 1)
			if err == nil || !strings.Contains(err.Error(), "unsupported image type") {
				t.Fatalf("expected unsupported image type error, got %v", err)
			}
			if called {
				t.Fatalf("repo must not be called for unsupported extension")
			}
		})
	}
}

func TestUploadBlogImage_RejectsContentMismatch(t *testing.T) {
	tests := []struct {
		name string
		file string
		data []byte
	}{
		{name: "png bytes as jpg", file: "a.jpg", data: pngBytes()},
		{name: "jpeg bytes as png", file: "a.png", data: jpegBytes()},
		{name: "html as png", file: "a.png", data: []byte("<html><body>hi</body></html>")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewImgService(&stubRepo{
				uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
				deleteFn: func(ctx context.Context, filePath string) error { return nil },
			})
			if _, err := svc.UploadBlogImage(context.Background(), tc.file, encode(tc.data), 1); err == nil {
				t.Fatalf("expected mismatch error")
			}
		})
	}
}

func TestUploadBlogImage_RejectsOversizedPayload(t *testing.T) {
	big := make([]byte, maxImageBytes+1)
	copy(big, pngBytes())
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
		deleteFn: func(ctx context.Context, filePath string) error { return nil },
	})
	_, err := svc.UploadBlogImage(context.Background(), "a.png", encode(big), 1)
	if err == nil || !strings.Contains(err.Error(), "maximum allowed size") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestUploadBlogImage_RepoError(t *testing.T) {
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error {
			return errors.New("repo error")
		},
		deleteFn: func(ctx context.Context, filePath string) error { return nil },
	})
	_, err := svc.UploadBlogImage(context.Background(), "a.png", encode(pngBytes()), 1)
	if err == nil || !strings.Contains(err.Error(), "repo error") {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestUploadBlogImage_ResponseSizeUsesEncodedDataLength(t *testing.T) {
	origUUID := generateUUID
	generateUUID = func() string { return "size-uuid" }
	t.Cleanup(func() { generateUUID = origUUID })

	encoded := encode(pngBytes())
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

func TestDeleteBlogImage_TrimsSingleLeadingSlash(t *testing.T) {
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
		deleteFn: func(ctx context.Context, filePath string) error {
			if filePath != "1/blog/img/a.png" {
				t.Fatalf("unexpected filePath %q", filePath)
			}
			return nil
		},
	})
	if err := svc.DeleteBlogImage(context.Background(), "/1/blog/img/a.png"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestDeleteBlogImage_RejectsInvalidPaths(t *testing.T) {
	invalid := []string{
		"",
		"/",
		"../1/blog/img/a.png",
		"1/blog/img/../../2/blog/img/a.png",
		`1\blog\img\a.png`,
		"http://evil/1/blog/img/a.png",
		"https://evil/a.png",
		"./1/blog/img/a.png",
		"//1/blog/img/a.png",
		"1/blog/a.png",
		"1/blog/img/",
		"abc/blog/img/a.png",
		"1/blog/img/a.png/extra",
	}
	for _, p := range invalid {
		t.Run(p, func(t *testing.T) {
			called := false
			svc := NewImgService(&stubRepo{
				uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
				deleteFn: func(ctx context.Context, filePath string) error {
					called = true
					return nil
				},
			})
			if err := svc.DeleteBlogImage(context.Background(), p); err == nil {
				t.Fatalf("expected invalid path error for %q", p)
			}
			if called {
				t.Fatalf("repo must not be called for invalid path %q", p)
			}
		})
	}
}

func TestDeleteBlogImage_RepoError(t *testing.T) {
	svc := NewImgService(&stubRepo{
		uploadFn: func(ctx context.Context, file []byte, filePath string, contentType string) error { return nil },
		deleteFn: func(ctx context.Context, filePath string) error { return errors.New("delete fail") },
	})
	if err := svc.DeleteBlogImage(context.Background(), "1/blog/img/a.png"); err == nil {
		t.Fatalf("expected error")
	}
}
