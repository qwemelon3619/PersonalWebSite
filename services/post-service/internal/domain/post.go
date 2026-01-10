package domain

import "time"

type Post struct {
	ID          uint       `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Content     string     `json:"content" db:"content"`
	AuthorID    uint       `json:"author_id" db:"author_id"`
	AuthorName  string     `json:"author_name,omitempty" db:"author_name"`
	Published   bool       `json:"published" db:"published"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type CreatePostRequest struct {
	Title     string `json:"title" binding:"required,min=1,max=200"`
	Content   string `json:"content" binding:"required"`
	Published bool   `json:"published"`
}

type UpdatePostRequest struct {
	Title     *string `json:"title,omitempty" binding:"omitempty,min=1,max=200"`
	Content   *string `json:"content,omitempty"`
	Published *bool   `json:"published,omitempty"`
}

type PostFilter struct {
	AuthorID  *uint   `json:"author_id"`
	Published *bool   `json:"published"`
	Limit     int     `json:"limit"`
	Offset    int     `json:"offset"`
	OrderBy   string  `json:"order_by"`
	Search    *string `json:"search"`
}

type PostRepository interface {
	Create(post *Post) error
	GetByID(id uint) (*Post, error)
	GetAll(filter PostFilter) ([]*Post, error)
	Update(post *Post) error
	Delete(id uint) error
	GetByAuthorID(authorID uint) ([]*Post, error)
}

type PostService interface {
	CreatePost(req CreatePostRequest, authorID uint, authorName string) (*Post, error)
	GetPost(id uint) (*Post, error)
	GetPostsByFilter(filter PostFilter) ([]*Post, error)
	UpdatePost(id uint, req UpdatePostRequest, authorID uint) (*Post, error)
	DeletePost(id, authorID uint) error
}
