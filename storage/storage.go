package storage

import (
	"bufio"
	"fmt"
	"github.com/KirillShapovalov/go_word_searcher/services/fileUtils"
	"os"
	"strings"
	"sync"
)

type FileManager struct {
	files []string
	mu    sync.Mutex
}

type IndexManager struct {
	Index map[string][]string
	Mu    sync.RWMutex
}

type FileStorage struct {
	FileManager  *FileManager
	IndexManager *IndexManager
}

func NewFileStorage() *FileStorage {
	return &FileStorage{
		FileManager:  &FileManager{files: []string{}},
		IndexManager: &IndexManager{Index: make(map[string][]string)},
	}
}

func (fm *FileManager) AddFile(filePath string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.files = append(fm.files, filePath)
}

func (fm *FileManager) GetFiles() []string {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	return append([]string{}, fm.files...)
}

func (fm *FileManager) ClearFiles() {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.files = []string{}
}

func (im *IndexManager) IndexFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for indexing: %w", err)
	}
	defer fileUtils.HandleDeferClose("index file", file.Close)

	scanner := bufio.NewScanner(file)
	localIndex := make(map[string]bool) // Используем локальный индекс для исключения дублирования слов в одном файле
	for scanner.Scan() {
		words := strings.Fields(scanner.Text())
		for _, word := range words {
			word = strings.ToLower(word) // Приводим слово к нижнему регистру
			if !localIndex[word] {
				localIndex[word] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error while reading file during indexing: %w", err)
	}

	// Обновляем глобальный индекс с использованием мьютекса
	im.Mu.Lock()
	defer im.Mu.Unlock()
	for word := range localIndex {
		im.Index[word] = append(im.Index[word], filePath)
	}

	return nil
}
