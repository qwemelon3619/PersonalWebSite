package handler

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/img-service/internal/service"
)

type uploadImageRequest struct {
	Filename string `json:"filename"`
	UserId   string `json:"userId"`
	Data     string `json:"data"` // base64 string
	MimeType string `json:"mimeType"`
}

func UploadImageHandler(svc *service.ImgService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req uploadImageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		// data: "data:image/png;base64,...."
		base64data := req.Data
		if idx := strings.Index(base64data, ","); idx != -1 {
			base64data = base64data[idx+1:]
		}
		imgBytes, err := base64.StdEncoding.DecodeString(base64data)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 data"})
			return
		}
		img, err := svc.UploadImage(c.Request.Context(), req.Filename, imgBytes, req.MimeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": img.URL, "name": img.Name, "size": img.Size, "mimeType": img.MimeType})
	}
}
