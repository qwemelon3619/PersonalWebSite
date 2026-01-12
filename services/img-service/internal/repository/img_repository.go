package repository

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/config"
)

type ImgRepository struct {
	BlobClient *azblob.Client
	config     config.BlobConfig
}

func NewImgRepository(blobClient *azblob.Client, config config.BlobConfig) *ImgRepository {
	return &ImgRepository{BlobClient: blobClient, config: config}
}
func (r *ImgRepository) UploadBlogImageToBlob(ctx context.Context, file []byte, filePath string, contentType string) error {
	if r.BlobClient == nil {
		return fmt.Errorf("Azure container client is nil")
	}
	resp, err := r.BlobClient.UploadBuffer(ctx, r.config.BlobContainerName, filePath, file, &azblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: to.Ptr(contentType),
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("UploadBlogImageToBlob response: %+v\n", resp)
	return nil
}
