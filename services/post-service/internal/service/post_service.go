package service

import (
	"fmt"

	"github.com/microcosm-cc/bluemonday"
	"seungpyo.lee/PersonalWebSite/pkg/logger"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/adapter"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/config"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/model"
)

type postService struct {
	postRepo     domain.PostRepository
	tagRepo      domain.TagRepository
	config       *config.PostConfig
	imageAdapter adapter.ImageAdapter
	transAdapter adapter.TranslationAdapter
	logger       *logger.Logger
}

// NewPostService creates a new PostService with the given repository.
func NewPostService(postRepo domain.PostRepository, tagRepo domain.TagRepository, config *config.PostConfig, imageAdapter adapter.ImageAdapter, transAdapter adapter.TranslationAdapter) domain.PostService {
	return &postService{postRepo: postRepo, tagRepo: tagRepo, config: config, imageAdapter: imageAdapter, transAdapter: transAdapter, logger: logger.New("info")}
}

// CreatePost creates a new blog post with the given request and author ID.
func (s *postService) CreatePost(req model.CreatePostRequest, authorID uint, authorName string) (*domain.Post, error) {
	// Process Markdown for image uploads BEFORE sanitization
	var processedContent string
	var err error
	processedContent, err = s.imageAdapter.ProcessMarkdownForImages(req.Content, authorID)
	if err != nil {
		return nil, fmt.Errorf("failed to process images in content: %w", err)
	}

	// Sanitize content
	p := bluemonday.UGCPolicy()
	safeContent := p.Sanitize(processedContent)

	var thumbnailURL string
	if req.ThumbnailData != "" {
		url, err := s.imageAdapter.UploadImage(req.ThumbnailData, authorID)
		if err != nil {
			return nil, fmt.Errorf("failed to upload thumbnail: %w", err)
		}
		thumbnailURL = url
	} else {
		thumbnailURL = "" // use blank if no data
	}

	post := &domain.Post{
		Title:      req.Title,
		Content:    safeContent,
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

	// Auto-translate if translation service configured
	if s.config != nil && s.config.TranslationAPIURL != "" {
		if t, err := s.transAdapter.TranslateSingle(post.Title); err == nil {
			post.EnTitle = t
			s.logger.Info(fmt.Sprintf("Translated title to: %s", t))
		} else {
			s.logger.Error(fmt.Sprintf("Failed to translate title: %v", err))
		}

		if t, err := s.transAdapter.TranslateMarkdown(processedContent); err == nil {
			post.EnContent = t
			s.logger.Info(fmt.Sprintf("Translated content to: %s", t))
		} else {
			s.logger.Error(fmt.Sprintf("Failed to translate content: %v", err))
		}
		// persist translations
		if err := s.postRepo.Update(post); err != nil {
			// log but don't fail creation
			s.logger.Error(fmt.Sprintf("Failed to persist translations: %v", err))
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
func (s *postService) GetPostsByFilter(filter model.PostFilter) ([]*domain.Post, error) {
	posts, err := s.postRepo.GetAll(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}
	return posts, nil
}

// UpdatePost updates an existing post if the author matches.
func (s *postService) UpdatePost(id uint, req model.UpdatePostRequest, authorID uint) (*domain.Post, error) {
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
	var safeContent string
	var processedContent string
	if req.Content != nil {
		// Process Markdown for image uploads BEFORE sanitization
		var err error
		processedContent, err = s.imageAdapter.ProcessMarkdownForImages(*req.Content, authorID)
		if err != nil {
			return nil, fmt.Errorf("failed to process images in content: %w", err)
		}
		p := bluemonday.UGCPolicy()
		safeContent = p.Sanitize(processedContent)
		post.Content = safeContent
	}
	if req.ThumbnailData != nil && *req.ThumbnailData != "" {
		url, err := s.imageAdapter.UploadImage(*req.ThumbnailData, authorID)
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

	// If translation service is configured, always try to translate
	if s.config != nil && s.config.TranslationAPIURL != "" {
		if req.Title != nil {
			if t, err := s.transAdapter.TranslateSingle(post.Title); err == nil {
				post.EnTitle = t
				s.logger.Info(fmt.Sprintf("Translated title to: %s", t))
			} else {
				s.logger.Error(fmt.Sprintf("Failed to translate title: %v", err))
			}
		}
		if req.Content != nil {
			if t, err := s.transAdapter.TranslateMarkdown(processedContent); err == nil {
				post.EnContent = t
				s.logger.Info(fmt.Sprintf("Translated content to: %s", t))
			} else {
				s.logger.Error(fmt.Sprintf("Failed to translate content: %v", err))
			}
		}
		// Persist translations if any were updated
		if err := s.postRepo.Update(post); err != nil {
			s.logger.Error(fmt.Sprintf("Failed to persist translations after update: %v", err))
		}
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
		if err := s.imageAdapter.DeleteImage(post.Thumbnail); err != nil {
			// Log error but proceed with post deletion
			fmt.Printf("Failed to delete thumbnail via img-service: %v\n", err)
		}
	} else {
		fmt.Println("No thumbnail to delete")
	}
	// Delete images in content via img-service if any
	imageURLs := s.imageAdapter.ExtractImageURLsFromContent(post.Content)
	for _, url := range imageURLs {
		if err := s.imageAdapter.DeleteImage(url); err != nil {
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
