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

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}
func main() {
	conf := config.LoadBlobConfig()
	client, err := azblob.NewClientFromConnectionString(conf.AzureStorageConnectionString, nil)
	handleError(err)
	//create container if not exists
	_, err = client.CreateContainer(context.TODO(), conf.BlobContainerName, &azblob.CreateContainerOptions{
		Access: to.Ptr(azblob.PublicAccessTypeBlob),
	})

	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.ErrorCode == string(bloberror.ContainerAlreadyExists) {
			//already exists,
			log.Printf("Container already exists, skipping creation.")
		} else {
			//other error
			handleError(err)
		}
	}

	imgageRepo := repository.NewImgRepository(client, *conf)
	imageService := service.NewImgService(imgageRepo)
	imageHandler := handler.NewBlogImageHandler(imageService)

	r := gin.Default()
	r.POST("/blog-image", imageHandler.UploadBlogImageHandler)
	r.DELETE("/blog-image", imageHandler.DeleteBlogImageHandler)

	if err := r.Run(":" + conf.ServerPort); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
