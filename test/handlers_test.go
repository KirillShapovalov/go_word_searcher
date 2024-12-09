package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/KirillShapovalov/go_word_searcher/internal/routes"
	"github.com/KirillShapovalov/go_word_searcher/services/upload"
	"github.com/KirillShapovalov/go_word_searcher/storage"
	"github.com/stretchr/testify/assert"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter(fileStorage *storage.FileStorage) *gin.Engine {
	r := gin.Default()
	routes.RegisterRoutes(r, fileStorage)
	return r
}

func TestUploadFile(t *testing.T) {
	s := storage.NewFileStorage()

	r := setupRouter(s)

	// Создание временного файла для теста
	fileContent := []byte("test content")
	fileName := "testfile.txt"
	tmpFile, err := os.CreateTemp("", fileName)
	if err != nil {
		t.Errorf("failed to create temporary file: %v", err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Errorf("failed to remove temporary file: %v", err)
		}
	}(tmpFile.Name()) // Удаление файла после теста

	if _, err := tmpFile.Write(fileContent); err != nil {
		t.Errorf("failed to write to temporary file: %v", err)
	}
	err = tmpFile.Close()
	if err != nil {
		t.Errorf("failed to close temporary file: %v", err)
	}

	// Создание запроса с файлом
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", fileName)
	_, err = part.Write(fileContent)
	if err != nil {
		t.Errorf("failed to write to temporary file: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Errorf("failed to close temporary file: %v", err)
	}

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверка, что запрос вернул статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code, "expected status 200, got %d, response: %s", w.Code, w.Body.String())

	// Проверка, что файл добавлен в хранилище
	files := s.FileManager.GetFiles()
	assert.Contains(t, files, upload.UploadDir+fileName, "Uploaded file not found in storage")

	// Проверка, что файл существует на диске
	uploadedFilePath := upload.UploadDir + fileName
	_, err = os.Stat(uploadedFilePath)
	assert.NoError(t, err, fmt.Sprintf("Uploaded file does not exist at path: %s", uploadedFilePath))

	// Очистка после теста
	err = os.Remove(uploadedFilePath)
	assert.NoError(t, err, fmt.Sprintf("Failed to remove uploaded file during cleanup: %s", uploadedFilePath))
}

func TestListFiles(t *testing.T) {
	s := storage.NewFileStorage()

	r := setupRouter(s)

	// Добавляем тестовые файлы в хранилище
	s.FileManager.AddFile("testfile1.txt")
	s.FileManager.AddFile("testfile2.txt")
	defer s.FileManager.ClearFiles() // Очищаем хранилище после теста

	req, _ := http.NewRequest("GET", "/files", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	expected := `{"files":["testfile1.txt","testfile2.txt"]}`
	if w.Body.String() != expected {
		t.Errorf("expected response %s, got %s", expected, w.Body.String())
	}
}

func TestSearchKeyword(t *testing.T) {
	s := storage.NewFileStorage()

	r := setupRouter(s)

	// Добавляем файлы в хранилище и создаём индекс
	s.FileManager.AddFile("testfile1.txt")
	s.FileManager.AddFile("testfile2.txt")
	defer s.FileManager.ClearFiles()

	s.IndexManager.Mu.Lock()
	s.IndexManager.Index["test"] = []string{"testfile1.txt"}
	s.IndexManager.Mu.Unlock()

	req, _ := http.NewRequest("GET", "/search?keyword=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	expected := `{"files":["testfile1.txt"]}`
	if w.Body.String() != expected {
		t.Errorf("expected response %s, got %s", expected, w.Body.String())
	}
}

func TestSearchKeywordFileError(t *testing.T) {
	s := storage.NewFileStorage()

	r := setupRouter(s)

	// Добавляем недоступный файл в хранилище
	s.FileManager.AddFile("non_existent_file.txt")
	defer s.FileManager.ClearFiles()

	// Выполняем поиск по ключевому слову
	req, _ := http.NewRequest("GET", "/search?keyword=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверка, что вернулся статус 400 Bad request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Проверка, что в ответе содержится сообщение об ошибке
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "error while searching", "Expected error message in response")
}

func TestSearchKeywordNotFound(t *testing.T) {
	s := storage.NewFileStorage()

	r := setupRouter(s)

	tmpFile, err := os.CreateTemp(upload.UploadDir, "testfile1.txt")
	if err != nil {
		t.Errorf("Failed to create temporary file: %v", err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Errorf("failed to remove temporary file: %v", err)
		}
	}(tmpFile.Name()) // Удаляем файл после теста

	// Записываем в файл данные
	_, err = tmpFile.WriteString("this is a test file content")
	if err != nil {
		t.Errorf("Failed to write to temporary file: %v", err)
	}
	err = tmpFile.Close()
	if err != nil {
		t.Errorf("Failed to close temporary file: %v", err)
	}

	// Добавляем файлы в хранилище, но не создаём индекс
	s.FileManager.AddFile(tmpFile.Name())
	defer s.FileManager.ClearFiles()

	s.IndexManager.Mu.Lock()
	s.IndexManager.Index["test"] = []string{tmpFile.Name()}
	s.IndexManager.Mu.Unlock()

	req, _ := http.NewRequest("GET", "/search?keyword=unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверка, что вернулся статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200, got %d, error: %s", w.Code, w.Body.String())

	// Проверка, что список файлов пуст
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Failed to parse JSON response")
	assert.Nil(t, response["files"], "Expected no files to be returned")
}
