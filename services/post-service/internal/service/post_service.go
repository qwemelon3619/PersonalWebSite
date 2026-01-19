package service

import (
	"fmt"

	"github.com/microcosm-cc/bluemonday"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
)

type postService struct {
	postRepo domain.PostRepository
	tagRepo  domain.TagRepository
}

// NewPostService creates a new PostService with the given repository.
func NewPostService(postRepo domain.PostRepository, tagRepo domain.TagRepository) domain.PostService {
	return &postService{postRepo: postRepo, tagRepo: tagRepo}
}

// CreatePost creates a new blog post with the given request and author ID.
func (s *postService) CreatePost(req domain.CreatePostRequest, authorID uint, authorName string) (*domain.Post, error) {
	// Sanitize user provided HTML content using an UGC policy
	p := bluemonday.UGCPolicy()
	safeContent := p.Sanitize(req.Content)

	post := &domain.Post{
		Title:      req.Title,
		Content:    safeContent,
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
		p := bluemonday.UGCPolicy()
		post.Content = p.Sanitize(*req.Content)
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
	if err := s.postRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}
	return nil
}

// ListTags returns all available tags.
func (s *postService) ListTags() ([]*domain.Tag, error) {
	return s.tagRepo.ListTags()
}
