package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
)

type postRepository struct {
	db *gorm.DB
}

// NewPostRepository creates a new PostRepository with the given GORM DB instance.
func NewPostRepository(db *gorm.DB) domain.PostRepository {
	return &postRepository{db: db}
}

// Create inserts a new post into the database.
func (r *postRepository) Create(post *domain.Post) error {
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now
	if post.Published {
		post.PublishedAt = &now
	} else {
		post.PublishedAt = nil
	}
	if err := r.db.Create(post).Error; err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}
	return nil
}

// GetByID retrieves a post by its ID from the database.
func (r *postRepository) GetByID(id uint) (*domain.Post, error) {
	var post domain.Post
	if err := r.db.Preload("Tags").First(&post, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("failed to get post: %w", err)
	}
	return &post, nil
}

// GetAll returns all posts matching the given filter.
func (r *postRepository) GetAll(filter domain.PostFilter) ([]*domain.Post, error) {
	var posts []*domain.Post
	query := r.db.Model(&domain.Post{}).Preload("Tags")
	if filter.AuthorID != nil {
		query = query.Where("author_id = ?", *filter.AuthorID)
	}
	if filter.Published != nil {
		query = query.Where("published = ?", *filter.Published)
	}
	if filter.Search != nil {
		like := "%" + *filter.Search + "%"
		query = query.Where("title ILIKE ? OR content ILIKE ?", like, like)
	}
	if filter.Tag != nil && *filter.Tag != "" {
		// join tags via post_tags to filter by tag name
		query = query.Joins("JOIN post_tags pt ON pt.post_id = posts.id").Joins("JOIN tags t ON t.id = pt.tag_id").Where("t.name = ?", *filter.Tag)
	}
	orderBy := "created_at DESC"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	query = query.Order(orderBy)
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if err := query.Find(&posts).Error; err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	return posts, nil
}

// Update updates an existing post in the database.
func (r *postRepository) Update(post *domain.Post) error {
	now := time.Now()
	post.UpdatedAt = now
	if post.Published && post.PublishedAt == nil {
		post.PublishedAt = &now
	} else if !post.Published {
		post.PublishedAt = nil
	}
	result := r.db.Model(post).Updates(map[string]interface{}{
		"title":        post.Title,
		"content":      post.Content,
		"published":    post.Published,
		"published_at": post.PublishedAt,
		"updated_at":   post.UpdatedAt,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to update post: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("post not found")
	}
	return nil
}

// Delete removes a post by its ID from the database.
func (r *postRepository) Delete(id uint) error {
	result := r.db.Delete(&domain.Post{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete post: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("post not found")
	}
	return nil
}

// GetByAuthorID returns all posts by a specific author.
func (r *postRepository) GetByAuthorID(authorID uint) ([]*domain.Post, error) {
	filter := domain.PostFilter{
		AuthorID: &authorID,
		OrderBy:  "created_at DESC",
	}
	return r.GetAll(filter)
}
