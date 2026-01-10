package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

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

func BlogListHandler(c *gin.Context) {
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	resp, err := http.Get(apiGatewayURL + "/api/v1/posts")
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to fetch posts:", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to fetch posts"})
		return
	}
	defer resp.Body.Close()
	var posts []Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Invalid post data"})
		return
	}
	// Pagination logic
	pageSize := 8
	page := 1
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}
	totalPosts := len(posts)
	totalPages := (totalPosts + pageSize - 1) / pageSize
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > totalPosts {
		start = totalPosts
	}
	if end > totalPosts {
		end = totalPosts
	}
	pagedPosts := posts[start:end]
	// pageNumbers slice
	pageNumbers := []int{}
	for i := 1; i <= totalPages; i++ {
		pageNumbers = append(pageNumbers, i)
	}
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}
	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	c.HTML(http.StatusOK, "blog-list.html", gin.H{
		"posts":       pagedPosts,
		"username":    username,
		"isLoggedIn":  isLoggedIn,
		"page":        page,
		"totalPages":  totalPages,
		"pageNumbers": pageNumbers,
		"prevPage":    prevPage,
		"nextPage":    nextPage,
	})
}

func BlogPostHandler(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	c.HTML(http.StatusOK, "blog-post.html", gin.H{
		"username":   username,
		"isLoggedIn": isLoggedIn,
	})
}

func BlogPostSaveHandler(c *gin.Context) {
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	title := c.PostForm("article-title")
	published := true

	content := c.PostForm("article-content")
	removeContentString := []string{"<select class=\"ql-ui\" contenteditable=\"false\"><option value=\"plain\">Plain</option><option value=\"bash\">Bash</option><option value=\"cpp\">C++</option><option value=\"cs\">C#</option><option value=\"css\">CSS</option><option value=\"diff\">Diff</option><option value=\"xml\">HTML/XML</option><option value=\"java\">Java</option><option value=\"javascript\">JavaScript</option><option value=\"markdown\">Markdown</option><option value=\"php\">PHP</option><option value=\"python\">Python</option><option value=\"ruby\">Ruby</option><option value=\"sql\">SQL</option></select>", "contenteditable=\"true\""}
	for _, value := range removeContentString {
		content = strings.Replace(content, value, "", -1)
	}

	accessToken, err := c.Cookie("access_token")
	if err != nil || accessToken == "" {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{"error": "Need to Login"})
		return
	}
	payload := map[string]interface{}{
		"title":     title,
		"content":   content,
		"published": published,
	}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", apiGatewayURL+"/api/v1/posts", bytes.NewReader(reqBody))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to save post"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errMsg map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errMsg)
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": errMsg["error"]})
		return
	}
	c.Redirect(http.StatusFound, "/blog")
}

func BlogArticleHandler(c *gin.Context) {
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	articleNumer := c.Param("articleNumber")

	resp, err := http.Get(apiGatewayURL + "/api/v1/posts/" + articleNumer)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to fetch posts:", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to fetch posts"})
		return
	}
	defer resp.Body.Close()
	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Invalid post data"})
		return
	}
	// Content를 string으로 보장
	contentStr := post.Content
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	c.HTML(http.StatusOK, "blog-article.html", gin.H{
		"post": gin.H{
			"ID":          post.ID,
			"Title":       post.Title,
			"Content":     contentStr,
			"AuthorID":    post.AuthorID,
			"AuthorName":  post.AuthorName,
			"Published":   post.Published,
			"PublishedAt": post.PublishedAt,
			"CreatedAt":   post.CreatedAt,
			"UpdatedAt":   post.UpdatedAt,
		},
		"username":   username,
		"isLoggedIn": isLoggedIn,
	})
}

func BlogEditHandler(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	articleNumber := c.Param("articleNumber")
	var post *Post = nil
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	resp, err := http.Get(apiGatewayURL + "/api/v1/posts/" + articleNumber)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var p Post
		if err := json.NewDecoder(resp.Body).Decode(&p); err == nil {
			post = &p
		}
	}
	if post == nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to fetch post for editing"})
		return
	}
	c.HTML(http.StatusOK, "blog-post.html", gin.H{
		"username":      username,
		"isLoggedIn":    isLoggedIn,
		"articleNumber": articleNumber,
		"post": gin.H{
			"ID":          post.ID,
			"Title":       post.Title,
			"Content":     post.Content,
			"AuthorID":    post.AuthorID,
			"AuthorName":  post.AuthorName,
			"Published":   post.Published,
			"PublishedAt": post.PublishedAt,
			"CreatedAt":   post.CreatedAt,
			"UpdatedAt":   post.UpdatedAt,
		}})
}

func BlogRemoveHandler(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	articleNumber := c.Param("articleNumber")
	c.HTML(http.StatusOK, "blog-remove.html", gin.H{
		"username":      username,
		"isLoggedIn":    isLoggedIn,
		"articleNumber": articleNumber,
	})
}
func BlogRemovingHandler(c *gin.Context) {
	username, err := c.Cookie("user")
	isLoggedIn := err == nil && username != ""
	if !isLoggedIn {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{"error": "Need to Login"})
		return
	}
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:8080"
	}
	postID := c.Param("articleNumber")
	accessToken, err := c.Cookie("access_token")
	if err != nil || accessToken == "" {
		c.HTML(http.StatusUnauthorized, "error.html", gin.H{"error": "Need to Login"})
		return
	}
	req, err := http.NewRequest("DELETE", apiGatewayURL+"/api/v1/posts/"+postID, nil)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Failed to delete post"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		var errMsg map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errMsg)
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": errMsg["error"]})
		return
	}
	c.Redirect(http.StatusFound, "/blog")
}
