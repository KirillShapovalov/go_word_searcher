package main

import (
	"github.com/KirillShapovalov/go_word_searcher/internal/routes"
	"github.com/KirillShapovalov/go_word_searcher/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	fileStorage := storage.NewFileStorage()
	routes.RegisterRoutes(r, fileStorage)
	err := r.Run(":8080")
	if err != nil {
		panic(err)
	}
}
