package domain

import (
	"context"
	"io"

	"seungpyo.lee/PersonalWebSite/services/img-service/internal/model"
)

// blog file url -> /{username}/blog/img/{imagename}

type ImgRepository interface {
	UploadImageToBlob(ctx context.Context, file io.Reader, filePath string, contentType string) (bool, error)
}

type ImgService interface {
	UploadImage(ctx context.Context, filename string, data string, userId int) (*model.ImageResponse, error)
}
