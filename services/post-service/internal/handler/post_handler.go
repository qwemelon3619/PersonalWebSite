package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/domain"
	"seungpyo.lee/PersonalWebSite/services/post-service/internal/model"
)

const (
	defaultPostLimit = 30
	maxPostLimit     = 100
)

// PostHandler handles HTTP requests for blog posts.
type PostHandler struct {
	Service domain.PostService
}

// NewPostHandler creates a new PostHandler.
func NewPostHandler(service domain.PostService) *PostHandler {
	return &PostHandler{Service: service}
}

// optionalCallerID returns the gateway-provided caller ID, or nil when the
// header is absent or malformed.
func optionalCallerID(c *gin.Context) *uint {
	userIDStr := c.GetHeader("X-User-Id")
	if userIDStr == "" {
		return nil
	}
	parsed, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return nil
	}
	userID := uint(parsed)
	return &userID
}

func isNotFoundError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}

// CreatePost handles POST /posts. Creates a new blog post.
func (h *PostHandler) CreatePost(c *gin.Context) {
	var req model.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	// passed jwt middleware, get user id from jwt claims
	userIDStr := c.GetHeader("X-User-Id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	parsed, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}
	userID := uint(parsed)
	post, err := h.Service.CreatePost(req, userID)
	if err != nil {
		log.Printf("failed to create post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create post"})
		return
	}
	c.JSON(http.StatusCreated, post)
}

// GetPost handles GET /posts/:id. Retrieves a single post by ID.
func (h *PostHandler) GetPost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	post, err := h.Service.GetPost(uint(id), optionalCallerID(c))
	if err != nil {
		log.Printf("failed to get post: %v", err)
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get post"})
		return
	}
	c.JSON(http.StatusOK, post)
}

// GetPosts handles GET /posts. Lists posts with optional filters.
func (h *PostHandler) GetPosts(c *gin.Context) {
	var filter model.PostFilter
	if authorIDStr := c.Query("author_id"); authorIDStr != "" {
		authorID, err := strconv.ParseUint(authorIDStr, 10, 64)
		if err == nil {
			authorIDUint := uint(authorID)
			filter.AuthorID = &authorIDUint
		}
	}
	callerID := optionalCallerID(c)
	if callerID == nil {
		// Anonymous (or malformed caller identity): never expose drafts.
		published := true
		filter.Published = &published
	} else if publishedStr := c.Query("published"); publishedStr != "" {
		if published, err := strconv.ParseBool(publishedStr); err == nil {
			filter.Published = &published
		}
	}

	// Clamp pagination so callers cannot request unbounded result sets.
	filter.Limit = defaultPostLimit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			if limit < 1 {
				limit = 1
			}
			if limit > maxPostLimit {
				limit = maxPostLimit
			}
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset > 0 {
			filter.Offset = offset
		}
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}
	if tag := c.Query("tag"); tag != "" {
		filter.Tag = &tag
	}
	posts, err := h.Service.GetPostsByFilter(filter, callerID)
	if err != nil {
		log.Printf("failed to list posts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list posts"})
		return
	}
	c.JSON(http.StatusOK, posts)
}

// GetTags handles GET /tags and returns all tags.
func (h *PostHandler) GetTags(c *gin.Context) {
	tags, err := h.Service.ListTags()
	if err != nil {
		log.Printf("failed to list tags: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tags"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

// UpdatePost handles PUT /posts/:id. Updates an existing post.
func (h *PostHandler) UpdatePost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req model.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	userIDStr := c.GetHeader("X-User-Id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	parsed, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}
	userID := uint(parsed)
	post, err := h.Service.UpdatePost(uint(id), req, userID)
	if err != nil {
		log.Printf("failed to update post: %v", err)
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "failed to update post"})
		return
	}
	c.JSON(http.StatusOK, post)
}

// DeletePost handles DELETE /posts/:id. Deletes a post by ID.
func (h *PostHandler) DeletePost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	userIDStr := c.GetHeader("X-User-Id")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	parsed, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}
	userID := uint(parsed)
	if err := h.Service.DeletePost(uint(id), userID); err != nil {
		log.Printf("failed to delete post: %v", err)
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "failed to delete post"})
		return
	}
	c.Status(http.StatusNoContent)
}
