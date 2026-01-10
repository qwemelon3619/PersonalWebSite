package service

import (
	"fmt"

	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
)

type postService struct {
	repo domain.PostRepository
}

// NewPostService creates a new PostService with the given repository.
func NewPostService(repo domain.PostRepository) domain.PostService {
	return &postService{repo: repo}
}

// CreatePost creates a new blog post with the given request and author ID.
func (s *postService) CreatePost(req domain.CreatePostRequest, authorID uint, authorName string) (*domain.Post, error) {
	post := &domain.Post{
		Title:      req.Title,
		Content:    req.Content,
		AuthorID:   authorID,
		Published:  req.Published,
		AuthorName: authorName,
	}
	if err := s.repo.Create(post); err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}
	return post, nil
}

// GetPost retrieves a post by its ID.
func (s *postService) GetPost(id uint) (*domain.Post, error) {
	post, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}
	return post, nil
}

// GetPostsByFilter returns a list of posts matching the given filter.
func (s *postService) GetPostsByFilter(filter domain.PostFilter) ([]*domain.Post, error) {
	posts, err := s.repo.GetAll(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}
	return posts, nil
}

// UpdatePost updates an existing post if the author matches.
func (s *postService) UpdatePost(id uint, req domain.UpdatePostRequest, authorID uint) (*domain.Post, error) {
	post, err := s.repo.GetByID(id)
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
		post.Content = *req.Content
	}
	if req.Published != nil {
		post.Published = *req.Published
	}
	if err := s.repo.Update(post); err != nil {
		return nil, fmt.Errorf("failed to update post: %w", err)
	}
	return post, nil
}

// DeletePost deletes a post if the author matches.
func (s *postService) DeletePost(id, authorID uint) error {
	post, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get post: %w", err)
	}
	if post.AuthorID != authorID {
		return fmt.Errorf("unauthorized: only the author can delete this post")
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}
	return nil
}
