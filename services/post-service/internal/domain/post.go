package domain

import (
	"time"

	"seungpyo.lee/PersonalWebSite/services/post-service/internal/model"
)

type Post struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title" gorm:"type:text;not null"`
	Content     string     `json:"content" gorm:"type:text;not null"`
	EnTitle     string     `json:"en_title,omitempty" gorm:"type:text"`
	EnContent   string     `json:"en_content,omitempty" gorm:"type:text"`
	Thumbnail   string     `json:"thumbnail,omitempty" gorm:"type:text"` // URL to thumbnail image
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

type PostRepository interface {
	Create(post *Post) error
	GetByID(id uint) (*Post, error)
	GetAll(filter model.PostFilter) ([]*Post, error)
	Update(post *Post) error
	Delete(id uint) error
	GetByAuthorID(authorID uint) ([]*Post, error)
}

type TagRepository interface {
	AttachTagsToPost(postID uint, tagNames []string) error
	ReplaceTagsForPost(postID uint, tagNames []string) error
	GetTagsForPost(postID uint) ([]*Tag, error)
	ListTags() ([]*Tag, error)
	DeleteTag(id uint) error
	DeleteUnusedTag(tagID uint) error
}

type PostService interface {
	CreatePost(req model.CreatePostRequest, authorID uint, authorName string) (*Post, error)
	GetPost(id uint) (*Post, error)
	GetPostsByFilter(filter model.PostFilter) ([]*Post, error)
	UpdatePost(id uint, req model.UpdatePostRequest, authorID uint) (*Post, error)
	DeletePost(id, authorID uint) error
	ListTags() ([]*Tag, error)
}
