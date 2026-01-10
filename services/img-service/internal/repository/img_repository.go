package repository

import (
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/config"
)

type ImgRepository struct {
	BlobClient *azblob.Client
	config     config.BlobConfig
}

func NewImgRepository(blobClient *azblob.Client, config config.BlobConfig) *ImgRepository {
	return &ImgRepository{BlobClient: blobClient, config: config}
}

func (r *ImgRepository) UploadImageToBlob(ctx context.Context, file io.Reader, fileName string, fileSize int64, mimeType string) (bool, error) {
	if r.BlobClient == nil {
		return true, fmt.Errorf("Azure container client is nil")
	}
	_, err := r.BlobClient.UploadStream(ctx, r.config.BlobContainerName, fileName, file, &azblob.UploadStreamOptions{
		BlockSize: int64(1024) * 256, // 256KB
	})
	if err != nil {
		return true, err
	}

	return true, nil
}
