package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/model"
)

type imageAdapterImpl struct {
	config *config.PostConfig
}

func NewImageAdapter(config *config.PostConfig) ImageAdapter {
	return &imageAdapterImpl{config: config}
}

func (a *imageAdapterImpl) UploadImage(data string, userID uint) (string, error) {
	// url := fmt.Sprintf("%s/api/v1/images", a.config.ApiGatewayURL)
	url := fmt.Sprintf("%s/blog-image", a.config.ImageServiceURL)
	reqBody := model.UploadImageRequest{
		Filename: "image.jpg", // dummy filename
		UserID:   fmt.Sprintf("%d", userID),
		Data:     data,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("img-service returned status %d", resp.StatusCode)
	}
	var imgResp model.UploadImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return "", err
	}
	return imgResp.URL, nil
}

func (a *imageAdapterImpl) DeleteImage(path string) error {
	// url := fmt.Sprintf("%s/api/v1/images", a.config.ApiGatewayURL)
	url := fmt.Sprintf("%s/blog-image", a.config.ImageServiceURL)
	reqBody := model.DeleteImageRequest{
		Path: path,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("img-service returned status %d", resp.StatusCode)
	}
	return nil
}

func (a *imageAdapterImpl) ProcessMarkdownForImages(content string, userID uint) (string, error) {
	// Use regex to find image markdown syntax with data URLs
	// Pattern: ![alt](data:...;base64,...)
	re := regexp.MustCompile(`!\[([^\]]*)\]\((data:[^;]+;base64,[^)]+)\)`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the data URL
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match // shouldn't happen
		}
		imageData := submatches[2]
		alt := submatches[1]

		// Upload the image
		uploadedURL, err := a.UploadImage(imageData, userID)
		if err != nil {
			// Log error but keep original data URL to avoid breaking content
			return match
		}
		// Replace with uploaded URL
		return fmt.Sprintf("![%s](%s)", alt, uploadedURL)
	}), nil
}

func (a *imageAdapterImpl) ExtractImageURLsFromContent(content string) []string {
	// Use regex to find image URLs in markdown: ![alt](url)
	re := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	matches := re.FindAllStringSubmatch(content, -1)
	var urls []string
	for _, match := range matches {
		if len(match) >= 3 {
			url := match[2]
			// Include HTTP URLs and relative paths, but not data URLs
			if strings.HasPrefix(url, "http") || (strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "data:")) {
				urls = append(urls, url)
			}
		}
	}
	return urls
}
