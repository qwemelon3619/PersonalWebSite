package domain

import (
	"context"
	"io"
)

// blog file url -> /{username}/blog/img/{imagename}
type ImageResponse struct {
	URL  string
	Name string
	Size int64
}

type ImgRepository interface {
	UploadImageToBlob(ctx context.Context, file io.Reader, filePath string, contentType string) (bool, error)
}

type ImgService interface {
	UploadImage(ctx context.Context, filename string, data string, userId int) (*ImageResponse, error)
}
