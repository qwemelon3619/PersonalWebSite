package domain

import "time"

type Post struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title" gorm:"type:text;not null"`
	Content     string     `json:"content" gorm:"type:text;not null"`
	AuthorID    uint       `json:"author_id" gorm:"index"`
	AuthorName  string     `json:"author_name,omitempty" gorm:"type:text"`
	Published   bool       `json:"published" gorm:"default:false"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Tags        []*Tag     `json:"tags,omitempty" gorm:"many2many:post_tags;constraint:OnDelete:CASCADE;"`
}

// Tag represents a post tag/label. Tags are normalized into their own table
// and associated with posts through a many-to-many join table.
type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"type:text;uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreatePostRequest struct {
	Title     string   `json:"title" binding:"required,min=1,max=200"`
	Content   string   `json:"content" binding:"required"`
	Published bool     `json:"published"`
	Tags      []string `json:"tags,omitempty"`
}

type UpdatePostRequest struct {
	Title     *string   `json:"title,omitempty" binding:"omitempty,min=1,max=200"`
	Content   *string   `json:"content,omitempty"`
	Published *bool     `json:"published,omitempty"`
	Tags      *[]string `json:"tags,omitempty"`
}

type PostFilter struct {
	AuthorID  *uint   `json:"author_id"`
	Published *bool   `json:"published"`
	Limit     int     `json:"limit"`
	Offset    int     `json:"offset"`
	OrderBy   string  `json:"order_by"`
	Search    *string `json:"search"`
	Tag       *string `json:"tag"`
}

type PostRepository interface {
	Create(post *Post) error
	GetByID(id uint) (*Post, error)
	GetAll(filter PostFilter) ([]*Post, error)
	Update(post *Post) error
	Delete(id uint) error
	GetByAuthorID(authorID uint) ([]*Post, error)
}

type TagRepository interface {
	AttachTagsToPost(postID uint, tagNames []string) error
	ReplaceTagsForPost(postID uint, tagNames []string) error
	GetTagsForPost(postID uint) ([]*Tag, error)
	ListTags() ([]*Tag, error)
}

type PostService interface {
	CreatePost(req CreatePostRequest, authorID uint, authorName string) (*Post, error)
	GetPost(id uint) (*Post, error)
	GetPostsByFilter(filter PostFilter) ([]*Post, error)
	UpdatePost(id uint, req UpdatePostRequest, authorID uint) (*Post, error)
	DeletePost(id, authorID uint) error
	ListTags() ([]*Tag, error)
}
