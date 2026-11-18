package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/model"
)

const (
	// maxImageBytes is the maximum decoded image size accepted for upload.
	maxImageBytes = 8 << 20
)

// allowedExtensions maps a lowercased client filename extension to the
// content type the decoded bytes must sniff as.
var allowedExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

var allowedSniffedTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// deletePathPattern matches the blob paths this service creates:
// "<userId>/blog/img/<filename>".
var deletePathPattern = regexp.MustCompile(`^\d+/blog/img/[\w.-]+$`)

type ImgRepo interface {
	UploadBlogImageToBlob(ctx context.Context, file []byte, filePath string, contentType string) error
	DeleteBlob(ctx context.Context, filePath string) error
}

type ImgService struct {
	Repo ImgRepo
}

func NewImgService(repo ImgRepo) *ImgService {
	return &ImgService{Repo: repo}
}

var generateUUID = func() string {
	return uuid.New().String()
}

func (s *ImgService) UploadBlogImage(ctx context.Context, filename string, data string, userId int) (*model.ImageResponse, error) {
	// data: "data:image/png;base64,...."
	if idx := strings.Index(data, ","); idx != -1 {
		data = data[idx+1:]
	}
	imgBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	if len(imgBytes) > maxImageBytes {
		return nil, fmt.Errorf("image exceeds maximum allowed size of %d bytes", maxImageBytes)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	expectedContentType, ok := allowedExtensions[ext]
	if !ok {
		return nil, fmt.Errorf("unsupported image type")
	}

	sniffed := http.DetectContentType(imgBytes)
	if !allowedSniffedTypes[sniffed] {
		return nil, fmt.Errorf("unsupported image type")
	}
	if !contentTypesMatch(expectedContentType, sniffed) {
		return nil, fmt.Errorf("image content does not match file extension")
	}

	uuidName := generateUUID() + ext
	imgPath := fmt.Sprintf("%d/blog/img/%s", userId, uuidName)
	if err := s.Repo.UploadBlogImageToBlob(ctx, imgBytes, imgPath, expectedContentType); err != nil {
		return nil, err
	}
	return &model.ImageResponse{
		URL:  imgPath,
		Name: filename,
		Size: int64(len(data)),
	}, nil
}

// contentTypesMatch reports whether the extension-implied type is consistent
// with the sniffed type. jpg/jpeg are both normalized to image/jpeg above.
func contentTypesMatch(expected, sniffed string) bool {
	return expected == sniffed
}

func (s *ImgService) DeleteBlogImage(ctx context.Context, filePath string) error {
	normalized, err := validateDeletePath(filePath)
	if err != nil {
		return err
	}
	return s.Repo.DeleteBlob(ctx, normalized)
}

// validateDeletePath rejects traversal attempts and paths that do not look
// like blobs created by this service. It returns the path with a single
// leading slash removed, matching the key format used by the repository.
func validateDeletePath(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("invalid image path")
	}
	if strings.Contains(filePath, "..") || strings.Contains(filePath, `\`) || strings.Contains(filePath, "://") {
		return "", fmt.Errorf("invalid image path")
	}
	if strings.HasPrefix(filePath, "./") {
		return "", fmt.Errorf("invalid image path")
	}
	path := strings.TrimPrefix(filePath, "/")
	if path == "" || strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("invalid image path")
	}
	if !deletePathPattern.MatchString(path) {
		return "", fmt.Errorf("invalid image path")
	}
	return path, nil
}
