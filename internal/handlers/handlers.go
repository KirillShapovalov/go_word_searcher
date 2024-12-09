package handlers

import (
	"fmt"
	"github.com/KirillShapovalov/go_word_searcher/services/fileUtils"
	"github.com/KirillShapovalov/go_word_searcher/services/search"
	"github.com/KirillShapovalov/go_word_searcher/services/upload"
	"github.com/KirillShapovalov/go_word_searcher/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Storage *storage.FileStorage
}

func NewHandlers(storage *storage.FileStorage) *Handlers {
	return &Handlers{
		Storage: storage,
	}
}

func (h *Handlers) UploadFile(c *gin.Context) {
	// Получаем файл и метаинформацию
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error while reading file": err.Error()})
		return
	}

	defer fileUtils.HandleDeferClose("source file", file.Close)

	// Сохраняем файл
	filePath, err := upload.SaveFile(file, header)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error while saving file": err.Error()})
		return
	}

	// Добавляем файл в хранилище
	h.Storage.FileManager.AddFile(filePath)

	// Индексируем содержимое файла
	go func() {
		if err := h.Storage.IndexManager.IndexFile(filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error while indexing file": err.Error()})
			return
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("file %s uploaded and indexed", header.Filename)})
}

func (h *Handlers) ListFiles(c *gin.Context) {
	files := h.Storage.FileManager.GetFiles()
	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *Handlers) SearchKeyword(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"files": nil,
			"error": "keyword is required",
		})
		return
	}

	files, err := search.FindWordInFiles(keyword, h.Storage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"files": nil,
			"error": "error while searching in files: " + err.Error(),
		})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusOK, gin.H{"files": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}
