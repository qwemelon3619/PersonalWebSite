package adapter

type ImageAdapter interface {
	UploadImage(data string, userID uint) (string, error)
	DeleteImage(path string) error
	ProcessMarkdownForImages(content string, userID uint) (string, error)
	ExtractImageURLsFromContent(content string) []string
}
