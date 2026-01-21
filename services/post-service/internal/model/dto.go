package model

// UploadImageRequest represents the request payload for uploading an image to img-service
type UploadImageRequest struct {
	Filename string `json:"filename"`
	UserID   string `json:"userId"`
	Data     string `json:"data"`
}

// DeleteImageRequest represents the request payload for deleting an image from img-service
type DeleteImageRequest struct {
	Path string `json:"path"`
}

// UploadImageResponse represents the response from img-service upload
type UploadImageResponse struct {
	URL string `json:"URL"`
}

// CreatePostRequest represents the request payload for creating a new post
type CreatePostRequest struct {
	Title         string   `json:"title" binding:"required,min=1,max=200"`
	Content       string   `json:"content" binding:"required"`
	ThumbnailData string   `json:"thumbnail_data,omitempty"`
	Published     bool     `json:"published"`
	Tags          []string `json:"tags,omitempty"`
}

// UpdatePostRequest represents the request payload for updating an existing post
type UpdatePostRequest struct {
	Title         *string   `json:"title,omitempty" binding:"omitempty,min=1,max=200"`
	Content       *string   `json:"content,omitempty"`
	ThumbnailData *string   `json:"thumbnail_data,omitempty"`
	Published     *bool     `json:"published,omitempty"`
	Tags          *[]string `json:"tags,omitempty"`
}

// PostFilter represents the filter criteria for querying posts
type PostFilter struct {
	AuthorID  *uint   `json:"author_id"`
	Published *bool   `json:"published"`
	Limit     int     `json:"limit"`
	Offset    int     `json:"offset"`
	OrderBy   string  `json:"order_by"`
	Search    *string `json:"search"`
	Tag       *string `json:"tag"`
}
