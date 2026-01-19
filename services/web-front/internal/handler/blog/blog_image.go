package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type uploadImageRequest struct {
	Filename string `json:"filename"`
	UserId   string `json:"userId"`
	Data     string `json:"data"` // base64 string
}
type ImageResponse struct {
	URL  string
	Name string
	Size int64
}

// Upload logic separated (improves readability and reusability)
func (h *postHandler) uploadBase64Image(imgSrc string, userID string, accessToken string) (string, string, error) {
	ext := ".png"
	if semi := strings.Index(imgSrc, ";"); semi != -1 && len(imgSrc) > 11 {
		mime := imgSrc[11:semi]
		switch mime {
		case "jpeg", "jpg":
			ext = ".jpg"
		case "gif":
			ext = ".gif"
		case "webp":
			ext = ".webp"
		case "svg+xml":
			ext = ".svg"
		}
	}

	apiImgService := os.Getenv("IMG_SERVICE_URL")
	if apiImgService == "" {
		apiImgService = "http://img-service:8083"
	}
	// generate unique filename
	fname := fmt.Sprintf("%s-%s%s", strconv.FormatInt(time.Now().UnixNano(), 10), strconv.FormatInt(time.Now().Unix()%100000, 10), ext)

	reqBody := uploadImageRequest{
		Filename: fname,
		UserId:   userID,
		Data:     imgSrc,
	}

	bodyBytes, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error
	var resp *http.Response
	// simple retry: 2 attempts
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest("POST", apiImgService+"/blog-image", bytes.NewReader(bodyBytes))
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if accessToken != "" {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
		resp, err = client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
		// non-2xx
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("img service returned %d: %s", resp.StatusCode, string(b))
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	if resp == nil {
		if lastErr == nil {
			lastErr = fmt.Errorf("failed to call img service")
		}
		return "", "", lastErr
	}
	defer resp.Body.Close()

	var imgResp ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return "", "", fmt.Errorf("failed to parse image upload response: %w", err)
	}
	if imgResp.URL == "" {
		return "", "", fmt.Errorf("empty url from img service")
	}
	blobBase := h.cfg.BlobBaseUrl
	// fallback default for local azurite
	return blobBase + "/" + imgResp.URL, imgResp.URL, nil
}

// processDeltaImages walks a Quill Delta structure and uploads any embedded base64 image
// values (insert.image == "data:image/..;base64,..."). It returns the modified delta object.
func (h *postHandler) processDeltaImages(delta interface{}, userID string, accessToken string) (interface{}, []string, error) {
	// locate ops array whether delta is {"ops": [...]} or already an array
	var ops []interface{}
	switch d := delta.(type) {
	case map[string]interface{}:
		if o, ok := d["ops"].([]interface{}); ok {
			ops = o
		}
	case []interface{}:
		ops = d
	default:
		// unknown structure — return as-is
		return delta, []string{}, nil
	}

	uploaded := []string{}
	for i, op := range ops {
		if opMap, ok := op.(map[string]interface{}); ok {
			if insert, ok := opMap["insert"]; ok {
				// insert can be string or map (for embeds like image)
				if insMap, ok := insert.(map[string]interface{}); ok {
					if imgVal, ok := insMap["image"].(string); ok {
						if strings.HasPrefix(imgVal, "data:image/") && strings.Contains(imgVal, ";base64,") {
							// upload and replace
							imgURL, blobPath, err := h.uploadBase64Image(imgVal, userID, accessToken)
							if err != nil {
								return nil, nil, err
							}
							insMap["image"] = imgURL
							// reassign
							opMap["insert"] = insMap
							ops[i] = opMap
							// collect uploaded path
							uploaded = append(uploaded, blobPath)
							// collect uploaded path
							uploaded = append(uploaded, blobPath)
						}
					}
				}
			}
		}
	}

	// If original delta was an object with ops, place updated ops back
	switch d := delta.(type) {
	case map[string]interface{}:
		d["ops"] = ops
		return d, uploaded, nil
	case []interface{}:
		return ops, uploaded, nil
	}
	return delta, uploaded, nil
}

// rollbackUploadedImages attempts to delete uploaded image paths from img-service.
func (h *postHandler) rollbackUploadedImages(paths []string, accessToken string) {
	if len(paths) == 0 {
		return
	}
	apiImgService := os.Getenv("IMG_SERVICE_URL")
	if apiImgService == "" {
		apiImgService = "http://img-service:8083"
	}
	client := &http.Client{Timeout: 10 * time.Second}
	for _, p := range paths {
		reqBody := map[string]string{"path": p}
		b, _ := json.Marshal(reqBody)
		req, err := http.NewRequest("DELETE", apiImgService+"/blog-image", bytes.NewReader(b))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if accessToken != "" {
			req.Header.Set("Authorization", "Bearer "+accessToken)
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
