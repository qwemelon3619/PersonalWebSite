package service

import (
	"bytes"
	"context"

	"seungpyo.lee/PersonalWebSite/services/img-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/repository"
)

type ImgService struct {
	Repo *repository.ImgRepository
}

func NewImgService(repo *repository.ImgRepository) *ImgService {
	return &ImgService{Repo: repo}
}

func (s *ImgService) UploadImage(ctx context.Context, filename string, data []byte, mimeType string) (*domain.Image, error) {
	_, err := s.Repo.UploadImageToBlob(ctx, bytes.NewReader(data), filename, int64(len(data)), mimeType)
	if err != nil {
		return nil, err
	}
	return &domain.Image{
		URL:      "sampleurl/" + filename,
		Name:     filename,
		Size:     int64(len(data)),
		MimeType: mimeType,
	}, nil
}
