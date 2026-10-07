package web

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vipthomps/gimle/backend/internal/api"
	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/prometheus"
)

// templFS - html templates
//
//go:embed templates/*
var templFS embed.FS

// pubFS - public folder
//
//go:embed public/*
var pubFS embed.FS

// Version - the version built into this binary, from public/version
func Version() string {
	file, err := pubFS.ReadFile("public/version")
	check.IfError(err)
	return strings.TrimSpace(strings.TrimPrefix(string(file), "VERSION="))
}

// Gui - start web server
func Gui() {
	const (
		colorCyan  = "\033[36m"
		colorReset = "\033[0m"
	)

	conf.AppConfig.Version = Version()

	address := conf.AppConfig.Host + ":" + conf.AppConfig.Port

	slog.Info(colorCyan + "\n=================================== " +
		"\n  Gimlé Version: " + conf.AppConfig.Version +
		"\n  Config dir: " + conf.AppConfig.DirPath +
		"\n  Default DB: " + conf.AppConfig.UseDB +
		"\n  Log level: " + conf.AppConfig.LogLevel +
		"\n  Web GUI: http://" + address +
		"\n=================================== " + colorReset)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	templ := template.Must(template.New("").ParseFS(templFS, "templates/*"))
	router.SetHTMLTemplate(templ) // templates

	router.StaticFS("/fs/", http.FS(pubFS)) // public

	router.GET("/", indexHandler)            // index.go
	router.GET("/config", indexHandler)      // index.go
	router.GET("/history", indexHandler)     // index.go
	router.GET("/host/*any", indexHandler)   // index.go
	router.GET("/hosts", indexHandler)       // index.go
	router.GET("/subnets", indexHandler)     // index.go
	router.GET("/subnet/*any", indexHandler) // index.go
	router.GET("/view/*any", indexHandler)   // index.go
	router.GET("/stats", indexHandler)       // index.go
	router.GET("/map", indexHandler)         // index.go
	router.GET("/bookmarks", indexHandler)   // index.go
	router.GET("/about", indexHandler)       // index.go
	router.GET("/docs", indexHandler)        // index.go
	router.GET("/docs/*any", indexHandler)   // index.go
	router.GET("/metrics", prometheus.Handler())

	api.Routes(router)

	err := router.Run(address)
	check.IfError(err)
}
