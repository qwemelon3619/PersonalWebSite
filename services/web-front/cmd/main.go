package main

import (
	"html/template"

	"github.com/gin-gonic/gin"
	"seungpyo.lee/PersonalWebSite/services/web-front/internal/config"
	auth "seungpyo.lee/PersonalWebSite/services/web-front/internal/handler/auth"
	blog "seungpyo.lee/PersonalWebSite/services/web-front/internal/handler/blog"
	page "seungpyo.lee/PersonalWebSite/services/web-front/internal/handler/page"
)

func mod(a, b int) int {
	return a % b
}

func main() {
	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"mod": mod,
		// renderSanitizedHTML: explicit helper used only for server-sanitized HTML
		"renderSanitizedHTML": func(s string) template.HTML { return template.HTML(s) },
	})
	r.LoadHTMLGlob("/app/services/web-front/templates/html/*.html")
	r.Static("/static", "/app/services/web-front/static")
	r.Static("/assets", "/app/services/web-front/templates/assets")

	cfg := config.LoadWebConfig()
	authH := auth.NewAuthHandler(cfg)
	blogH := blog.NewBlogHandler(cfg)
	postH := blog.NewPostHandler(cfg)
	pageH := page.NewPageHandler(cfg)

	r.GET("/", pageH.Index)
	r.GET("/about", pageH.About)

	r.GET("/login", authH.Login)
	r.POST("/login", authH.LoginPost)
	r.GET("/register", authH.Register)
	r.POST("/register", authH.RegisterPost)
	r.GET("/logout", authH.Logout)

	r.GET("/blog", blogH.List)
	r.GET("/blog-post", blogH.EditOrNew)
	r.GET("/blog-edit/:articleNumber", blogH.EditOrNew)
	r.POST("/blog-post", postH.Save)
	r.GET("/blog/:articleNumber", blogH.Article)
	r.GET("/blog-remove/:articleNumber", blogH.RemovePage)
	r.POST("/blog-remove/:articleNumber", blogH.Remove)

	r.GET("/contact", pageH.Contact)
	r.GET("/opensource", pageH.OpenSource)
	r.GET("/error", pageH.Error)

	r.Run(":3000")
}
