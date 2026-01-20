package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/service"
)

type imageHandler struct {
	service service.ImgService
}

type uploadImageRequest struct {
	Filename string `json:"filename"` //relative path
	UserId   string `json:"userId"`
	Data     string `json:"data"` // base64 string
}

func NewBlogImageHandler(service *service.ImgService) *imageHandler {
	return &imageHandler{service: *service}
}

func (h *imageHandler) UploadBlogImageHandler(c *gin.Context) {
	var req uploadImageRequest
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, img)

}

type deleteImageRequest struct {
	Path string `json:"path"` //relative path eg) /1/blog/img/uuid.jpg
}

func (h *imageHandler) DeleteBlogImageHandler(c *gin.Context) {
	var req deleteImageRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := h.service.DeleteBlogImage(c.Request.Context(), req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
