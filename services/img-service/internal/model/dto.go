package model

// ImageResponse represents the response from image upload
type ImageResponse struct {
	URL  string
	Name string
	Size int64
}

// UploadImageRequest represents the request payload for uploading an image
type UploadImageRequest struct {
	Filename string `json:"filename"` //relative path
	UserId   string `json:"userId"`
	Data     string `json:"data"` // base64 string
}

// DeleteImageRequest represents the request payload for deleting an image
type DeleteImageRequest struct {
	Path string `json:"path"` //relative path eg) /1/blog/img/uuid.jpg
}
