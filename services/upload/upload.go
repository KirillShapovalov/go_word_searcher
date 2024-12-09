package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/KirillShapovalov/go_word_searcher/services/fileUtils"
)

// UploadDir Путь к папке для хранения файлов
const UploadDir = "./uploads/"

func SaveFile(src multipart.File, header *multipart.FileHeader) (string, error) {
	// Создаём папку, если её нет
	if err := os.MkdirAll(UploadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Генерируем путь для сохранения файла
	filePath := UploadDir + header.Filename
	for i := 1; ; i++ {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			break // Если файл не существует, выходим из цикла
		}
		filePath = filepath.Join(UploadDir, fmt.Sprintf("%d_%s", i, header.Filename))
	}

	// Создаём целевой файл на диске
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer fileUtils.HandleDeferClose("destination file", dst.Close)

	// Копируем содержимое загруженного файла в целевой файл
	if _, err = io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	return filePath, nil
}
