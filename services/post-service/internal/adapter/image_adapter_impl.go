package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
	url := fmt.Sprintf("%s/api/v1/images", a.config.ApiGatewayURL)
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
	url := fmt.Sprintf("%s/api/v1/images", a.config.ApiGatewayURL)
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

func (a *imageAdapterImpl) ProcessDeltaForImages(content string, userID uint) (string, error) {
	var delta map[string]interface{}
	if err := json.Unmarshal([]byte(content), &delta); err != nil {
		// Not JSON, return as is
		return content, nil
	}
	ops, ok := delta["ops"].([]interface{})
	if !ok {
		return content, nil
	}
	for i, op := range ops {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			continue
		}
		insert, ok := opMap["insert"]
		if !ok {
			continue
		}
		if insertMap, ok := insert.(map[string]interface{}); ok {
			if imageData, ok := insertMap["image"].(string); ok && strings.HasPrefix(imageData, "data:") {
				// Upload the data URL image
				uploadedURL, err := a.UploadImage(imageData, userID)
				if err != nil {
					return "", fmt.Errorf("failed to upload image: %w", err)
				}
				// Replace with URL
				insertMap["image"] = uploadedURL
				ops[i] = opMap
			}
		}
	}
	// Marshal back to JSON
	updated, err := json.Marshal(delta)
	if err != nil {
		return "", err
	}
	return string(updated), nil
}

func (a *imageAdapterImpl) ExtractImageURLsFromContent(content string) []string {
	var delta map[string]interface{}
	if err := json.Unmarshal([]byte(content), &delta); err != nil {
		return nil
	}
	ops, ok := delta["ops"].([]interface{})
	if !ok {
		return nil
	}
	var urls []string
	for _, op := range ops {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			continue
		}
		insert, ok := opMap["insert"]
		if !ok {
			continue
		}
		if insertMap, ok := insert.(map[string]interface{}); ok {
			if imageURL, ok := insertMap["image"].(string); ok && imageURL != "" {
				// Include both HTTP URLs and relative paths
				if strings.HasPrefix(imageURL, "http") || strings.HasPrefix(imageURL, "/") {
					urls = append(urls, imageURL)
				}
			}
		}
	}
	return urls
}
