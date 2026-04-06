package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/model"
)

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
	return &model.ImageResponse{
		URL:  imgPath,
		Name: filename,
		Size: int64(len(data)),
	}, nil
}

func (s *ImgService) DeleteBlogImage(ctx context.Context, filePath string) error {
	return s.Repo.DeleteBlob(ctx, filePath)
}
