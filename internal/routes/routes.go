package routes

import (
	"github.com/KirillShapovalov/go_word_searcher/internal/handlers"
	"github.com/KirillShapovalov/go_word_searcher/storage"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, fileStorage *storage.FileStorage) {
	h := handlers.NewHandlers(fileStorage)

	r.POST("/upload", h.UploadFile)
	r.GET("/files", h.ListFiles)
	r.GET("/search", h.SearchKeyword)
}
