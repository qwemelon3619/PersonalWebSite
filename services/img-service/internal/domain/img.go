package domain

import (
	"context"
	"io"
)

// blog file url -> /{username}/blog/images/{imagename}
type Image struct {
	URL      string
	Name     string
	Size     int64
	MimeType string
}

type ImgRepository interface {
	UploadImageToBlob(ctx context.Context, file io.Reader, fileName string, fileSize int64, mimeType string) (bool, error)
}

type ImgService interface {
	UploadImage(ctx context.Context, filename string, data []byte, mimeType string) (*Image, error)
}
