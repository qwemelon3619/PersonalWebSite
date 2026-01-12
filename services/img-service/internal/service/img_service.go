package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/repository"
)

type ImgService struct {
	Repo *repository.ImgRepository
}

func NewImgService(repo *repository.ImgRepository) *ImgService {
	return &ImgService{Repo: repo}
}

func (s *ImgService) UploadBlogImage(ctx context.Context, filename string, data string, userId int) (*domain.ImageResponse, error) {
	// data: "data:image/png;base64,...."
	if idx := strings.Index(data, ","); idx != -1 {
		data = data[idx+1:]
	}
	imgBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	ext := ""
	if dot := strings.LastIndex(filename, "."); dot != -1 {
		ext = filename[dot:]
	}
	uuidName := generateUUID() + ext
	imgPath := fmt.Sprintf("%d/blog/img/%s", userId, uuidName)
	contentType := "image/png"
	switch ext {
	case ".jpeg":
		contentType = "image/jpeg"
	case ".jpg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}
	err = s.Repo.UploadBlogImageToBlob(ctx, imgBytes, imgPath, contentType)
	if err != nil {
		return nil, err
	}
	return &domain.ImageResponse{
		URL:  imgPath,
		Name: filename,
		Size: int64(len(data)),
	}, nil
}
func generateUUID() string {
	return uuid.New().String()
}
