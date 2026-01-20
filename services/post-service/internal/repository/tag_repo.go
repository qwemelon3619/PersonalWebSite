package repository

import (
	"fmt"

	"gorm.io/gorm"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
)

type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository creates a new TagRepository with the given GORM DB instance.
func NewTagRepository(db *gorm.DB) domain.TagRepository {
	return &tagRepository{db: db}
}

// AttachTagsToPost ensures tags exist and associates them with the given post ID.
func (r *tagRepository) AttachTagsToPost(postID uint, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}
	// Use GORM to FirstOrCreate each tag and append association
	tx := r.db.Begin()
	var post domain.Post
	if err := tx.First(&post, postID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("post not found: %w", err)
	}
	var tags []*domain.Tag
	for _, name := range tagNames {
		if name == "" {
			continue
		}
		var t domain.Tag
		if err := tx.Where("name = ?", name).FirstOrCreate(&t, domain.Tag{Name: name}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert tag %s: %w", name, err)
		}
		tags = append(tags, &t)
	}
	if len(tags) > 0 {
		if err := tx.Model(&post).Association("Tags").Append(tags); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to append tags to post: %w", err)
		}
	}
	return tx.Commit().Error
}

// ReplaceTagsForPost removes existing tag associations and attaches provided tags.
func (r *tagRepository) ReplaceTagsForPost(postID uint, tagNames []string) error {
	tx := r.db.Begin()
	var post domain.Post
	if err := tx.First(&post, postID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("post not found: %w", err)
	}
	var tags []*domain.Tag
	for _, name := range tagNames {
		if name == "" {
			continue
		}
		var t domain.Tag
		if err := tx.Where("name = ?", name).FirstOrCreate(&t, domain.Tag{Name: name}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to upsert tag %s: %w", name, err)
		}
		tags = append(tags, &t)
	}
	if err := tx.Model(&post).Association("Tags").Replace(tags); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to replace tags for post: %w", err)
	}
	return tx.Commit().Error
}

// GetTagsForPost returns tags attached to a post.
func (r *tagRepository) GetTagsForPost(postID uint) ([]*domain.Tag, error) {
	var post domain.Post
	if err := r.db.Preload("Tags").First(&post, postID).Error; err != nil {
		return nil, fmt.Errorf("failed to load post tags: %w", err)
	}
	return post.Tags, nil
}

// ListTags returns all tags ordered by name.
func (r *tagRepository) ListTags() ([]*domain.Tag, error) {
	var tags []*domain.Tag
	if err := r.db.Order("name ASC").Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, nil
}

// DeleteTag deletes a tag by its ID.
func (r *tagRepository) DeleteTag(id uint) error {
	if err := r.db.Delete(&domain.Tag{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	return nil
}

// DeleteUnusedTag deletes the tag if it is not associated with any posts.
func (r *tagRepository) DeleteUnusedTag(tagID uint) error {
	var count int64
	if err := r.db.Model(&domain.Post{}).Joins("JOIN post_tags ON posts.id = post_tags.post_id").Where("post_tags.tag_id = ?", tagID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count tag usage: %w", err)
	}
	if count == 0 {
		if err := r.db.Delete(&domain.Tag{}, tagID).Error; err != nil {
			return fmt.Errorf("failed to delete unused tag: %w", err)
		}
	}
	return nil
}
