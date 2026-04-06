package repository

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/config"
)

type BlobClient interface {
	UploadBuffer(ctx context.Context, containerName string, blobName string, data []byte, o *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error)
	DeleteBlob(ctx context.Context, containerName string, blobName string, o *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error)
}

type ImgRepository struct {
	BlobClient BlobClient
	config     config.BlobConfig
}

func NewImgRepository(blobClient BlobClient, config config.BlobConfig) *ImgRepository {
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

func (r *ImgRepository) DeleteBlob(ctx context.Context, filePath string) error {
	if r.BlobClient == nil {
		return fmt.Errorf("Azure container client is nil")
	}
	resp, err := r.BlobClient.DeleteBlob(ctx, r.config.BlobContainerName, filePath, nil)
	if err != nil {
		return err
	}
	fmt.Printf("DeleteBlob response: %+v\n", resp)
	return nil
}
