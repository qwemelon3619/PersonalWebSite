package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
)

type postService struct {
	postRepo domain.PostRepository
	tagRepo  domain.TagRepository
	config   *config.PostConfig
}

// NewPostService creates a new PostService with the given repository.
func NewPostService(postRepo domain.PostRepository, tagRepo domain.TagRepository, config *config.PostConfig) domain.PostService {
	return &postService{postRepo: postRepo, tagRepo: tagRepo, config: config}
}

// CreatePost creates a new blog post with the given request and author ID.
func (s *postService) CreatePost(req domain.CreatePostRequest, authorID uint, authorName string) (*domain.Post, error) {
	// Process Delta JSON for image uploads BEFORE sanitization
	var processedContent string
	var err error
	processedContent, err = s.processDeltaForImages(req.Content, authorID)
	if err != nil {
		return nil, fmt.Errorf("failed to process images in content: %w", err)
	}

	var thumbnailURL string
	if req.ThumbnailData != "" {
		url, err := s.callImgServiceToUpload(req.ThumbnailData, authorID)
		if err != nil {
			return nil, fmt.Errorf("failed to upload thumbnail: %w", err)
		}
		thumbnailURL = url
	} else {
		thumbnailURL = "" // use blank if no data
	}

	post := &domain.Post{
		Title:      req.Title,
		Content:    processedContent,
		Thumbnail:  thumbnailURL,
		AuthorID:   authorID,
		Published:  req.Published,
		AuthorName: authorName,
	}
	if err := s.postRepo.Create(post); err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	// Attach tags if provided (best-effort). Normalization is handled by repository.
	if len(req.Tags) > 0 {
		if err := s.tagRepo.AttachTagsToPost(post.ID, req.Tags); err != nil {
			return nil, fmt.Errorf("failed to attach tags: %w", err)
		}
	}
	return post, nil
}

// GetPost retrieves a post by its ID.
func (s *postService) GetPost(id uint) (*domain.Post, error) {
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}
	// Load tags for the post if repository supports it
	if tags, err := s.tagRepo.GetTagsForPost(id); err == nil {
		post.Tags = tags
	}
	return post, nil
}

// GetPostsByFilter returns a list of posts matching the given filter.
func (s *postService) GetPostsByFilter(filter domain.PostFilter) ([]*domain.Post, error) {
	posts, err := s.postRepo.GetAll(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}
	return posts, nil
}

// UpdatePost updates an existing post if the author matches.
func (s *postService) UpdatePost(id uint, req domain.UpdatePostRequest, authorID uint) (*domain.Post, error) {
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}
	if post.AuthorID != authorID {
		return nil, fmt.Errorf("unauthorized: only the author can update this post")
	}
	if req.Title != nil {
		post.Title = *req.Title
	}
	if req.Content != nil {
		// Process Delta JSON for image uploads BEFORE sanitization
		processedContent, err := s.processDeltaForImages(*req.Content, authorID)
		if err != nil {
			return nil, fmt.Errorf("failed to process images in content: %w", err)
		}
		p := bluemonday.UGCPolicy()
		safeContent := p.Sanitize(processedContent)
		post.Content = safeContent
	}
	if req.ThumbnailData != nil && *req.ThumbnailData != "" {
		url, err := s.callImgServiceToUpload(*req.ThumbnailData, authorID)
		if err != nil {
			return nil, fmt.Errorf("failed to upload thumbnail: %w", err)
		}
		post.Thumbnail = url
	}
	if req.Published != nil {
		post.Published = *req.Published
	}
	if err := s.postRepo.Update(post); err != nil {
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	// Replace tags if provided
	if req.Tags != nil {
		// if pointer to slice provided: replace existing tags
		if err := s.tagRepo.ReplaceTagsForPost(id, *req.Tags); err != nil {
			return nil, fmt.Errorf("failed to replace tags: %w", err)
		}
	}
	return post, nil
}

// DeletePost deletes a post if the author matches.
func (s *postService) DeletePost(id, authorID uint) error {
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get post: %w", err)
	}
	if post.AuthorID != authorID {
		return fmt.Errorf("unauthorized: only the author can delete this post")
	}
	// Store tags before deletion
	tags := post.Tags
	if err := s.postRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}
	// Delete unused tags
	for _, tag := range tags {
		if err := s.tagRepo.DeleteUnusedTag(tag.ID); err != nil {
			// Log error but proceed
			fmt.Printf("Failed to delete unused tag %s: %v\n", tag.Name, err)
		}
	}
	// Delete thumbnail image via img-service if exists
	if post.Thumbnail != "" {
		fmt.Printf("Deleting thumbnail: %s\n", post.Thumbnail)
		if err := s.callImgServiceToDelete(post.Thumbnail); err != nil {
			// Log error but proceed with post deletion
			fmt.Printf("Failed to delete thumbnail via img-service: %v\n", err)
		}
	} else {
		fmt.Println("No thumbnail to delete")
	}
	// Delete images in content via img-service if any
	imageURLs := s.extractImageURLsFromContent(post.Content)
	fmt.Printf("Extracted image URLs: %v\n", imageURLs)
	for _, url := range imageURLs {
		fmt.Printf("Deleting content image: %s\n", url)
		if err := s.callImgServiceToDelete(url); err != nil {
			// Log error but proceed
			fmt.Printf("Failed to delete content image via img-service: %v\n", err)
		}
	}

	return nil
}

// ListTags returns all available tags.
func (s *postService) ListTags() ([]*domain.Tag, error) {
	return s.tagRepo.ListTags()
}

// processDeltaForImages processes Quill Delta JSON, uploads data URL images, and replaces with URLs.
func (s *postService) processDeltaForImages(content string, userID uint) (string, error) {
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
				uploadedURL, err := s.callImgServiceToUpload(imageData, userID)
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

// extractImageURLsFromContent parses Quill Delta JSON and extracts image URLs.
func (s *postService) extractImageURLsFromContent(content string) []string {
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

// callImgServiceToUpload sends a upload request to img-service via api-gateway and returns the URL.
func (s *postService) callImgServiceToUpload(data string, userID uint) (string, error) {
	url := fmt.Sprintf("%s/api/v1/images", s.config.ApiGatewayURL) //upload endpoint
	fmt.Printf("Calling img-service to upload: %s\n", url)
	reqBody := map[string]interface{}{
		"filename": "thumbnail.jpg", // dummy filename
		"userId":   fmt.Sprintf("%d", userID),
		"data":     data,
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
	var imgResp struct {
		URL string `json:"URL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return "", err
	}
	fmt.Printf("Successfully uploaded thumbnail: %s\n", imgResp.URL)
	return imgResp.URL, nil
}

// callImgServiceToDelete sends a delete request to img-service via api-gateway for the given filename.
func (s *postService) callImgServiceToDelete(filename string) error {
	// Assume filename is already a relative path (e.g., /1/blog/img/uuid.jpg)
	// No URL conversion needed with nginx proxy
	url := fmt.Sprintf("%s/api/v1/images", s.config.ApiGatewayURL)
	fmt.Printf("Calling img-service to delete: %s\n", url)
	reqBody := map[string]string{"path": filename} //relative path
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
	fmt.Printf("Successfully deleted image: %s\n", filename)
	return nil
}
