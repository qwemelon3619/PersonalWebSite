package main

import (
	"context"
	"errors"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/handler"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/repository"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/service"
)

type blogImageHandler interface {
	UploadBlogImageHandler(c *gin.Context)
	DeleteBlogImageHandler(c *gin.Context)
}

type blobContainerClient interface {
	CreateContainer(ctx context.Context, containerName string, o *azblob.CreateContainerOptions) (azblob.CreateContainerResponse, error)
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func registerRoutes(r *gin.Engine, h blogImageHandler) {
	r.POST("/blog-image", h.UploadBlogImageHandler)
	r.DELETE("/blog-image", h.DeleteBlogImageHandler)
}

func ensureContainerExists(client blobContainerClient, containerName string) error {
	_, err := client.CreateContainer(context.TODO(), containerName, &azblob.CreateContainerOptions{
		Access: to.Ptr(azblob.PublicAccessTypeBlob),
	})
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.ErrorCode == string(bloberror.ContainerAlreadyExists) {
			log.Printf("Container already exists, skipping creation.")
			return nil
		}
		return err
	}
	return nil
}

func main() {
	conf := config.LoadBlobConfig()
	client, err := azblob.NewClientFromConnectionString(conf.AzureStorageConnectionString, nil)
	handleError(err)
	handleError(ensureContainerExists(client, conf.BlobContainerName))

	imgageRepo := repository.NewImgRepository(client, *conf)
	imageService := service.NewImgService(imgageRepo)
	imageHandler := handler.NewBlogImageHandler(imageService)

	r := gin.Default()
	registerRoutes(r, imageHandler)

	if err := r.Run(":" + conf.ServerPort); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
