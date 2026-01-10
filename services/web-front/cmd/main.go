package main

import (
	"html/template"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/web-front/internal/handler"
)

func mod(a, b int) int {
	return a % b
}

func main() {
	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"mod":      mod,
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	})
	r.LoadHTMLGlob("/app/services/web-front/templates/html/*.html")
	r.Static("/static", "/app/services/web-front/static")
	r.Static("/assets", "/app/services/web-front/templates/assets")

	r.GET("/", handler.IndexHandler)
	r.GET("/about", handler.AboutHandler)
	r.GET("/login", handler.LoginHandler)
	r.POST("/login", handler.LoginPostHandler)
	r.GET("/register", handler.RegisterHandler)
	r.POST("/register", handler.RegisterPostHandler)
	r.GET("/logout", handler.LogoutHandler)

	r.GET("/blog", handler.BlogListHandler)
	r.GET("/blog-post", handler.BlogPostHandler)
	r.POST("/blog-post", handler.BlogPostSaveHandler)
	r.GET("/blog/:articleNumber", handler.BlogArticleHandler)
	r.GET("/blog-edit/:articleNumber", handler.BlogEditHandler)
	r.GET("/blog-remove/:articleNumber", handler.BlogRemoveHandler)
	r.POST("/blog-remove/:articleNumber", handler.BlogRemovingHandler)
	r.GET("/contact", handler.ContactHandler)
	r.GET("/opensource", handler.OpenSourceHandler)
	r.GET("/error", handler.ErrorHandler)
	r.GET("/redirect", handler.RedirectHandler)

	r.Run(":3000")
}
