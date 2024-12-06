package storage

import (
	"bufio"
	"fmt"
	"github.com/KirillShapovalov/go_word_searcher/services/utils"
	"os"
	"strings"
	"sync"
)

type FileStorage struct {
	files []string
	Index map[string][]string
	mu    sync.Mutex
	IdxMu sync.RWMutex
}

func NewFileStorage() *FileStorage {
	return &FileStorage{
		files: []string{},
		Index: make(map[string][]string),
	}
}

func (s *FileStorage) AddFile(filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = append(s.files, filePath)
}

func (s *FileStorage) GetFiles() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.files...)
}

func (s *FileStorage) ClearFiles() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = []string{}
}

func (s *FileStorage) IndexFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for indexing: %w", err)
	}
	defer utils.HandleDeferClose("index file", file.Close)

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
	s.IdxMu.Lock()
	defer s.IdxMu.Unlock()
	for word := range localIndex {
		s.Index[word] = append(s.Index[word], filePath)
	}

	return nil
}
