package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/model"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/service"
)

// maxUploadBodyBytes caps the raw request body for image uploads (base64
// overhead included) before it is read into memory.
const maxUploadBodyBytes = 12 << 20

type BlogImageService interface {
	UploadBlogImage(ctx context.Context, filename string, data string, userId int) (*model.ImageResponse, error)
	DeleteBlogImage(ctx context.Context, filePath string) error
}

type imageHandler struct {
	service BlogImageService
}

func NewBlogImageHandler(service *service.ImgService) *imageHandler {
	return &imageHandler{service: service}
}

func (h *imageHandler) UploadBlogImageHandler(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBodyBytes)

	var req model.UploadImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	userId_I, err := strconv.Atoi(req.UserId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	img, err := h.service.UploadBlogImage(c.Request.Context(), req.Filename, req.Data, userId_I)
	if err != nil {
		log.Printf("failed to upload image: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload image"})
		return
	}
	c.JSON(http.StatusOK, img)

}

func (h *imageHandler) DeleteBlogImageHandler(c *gin.Context) {
	var req model.DeleteImageRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := h.service.DeleteBlogImage(c.Request.Context(), req.Path); err != nil {
		log.Printf("failed to delete image: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete image"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
