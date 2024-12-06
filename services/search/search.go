package search

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/KirillShapovalov/go_word_searcher/services/utils"
	"github.com/KirillShapovalov/go_word_searcher/storage"
)

func FindWordInFiles(keyword string, fileStorage *storage.FileStorage) ([]string, error) {
	// Получаем файлы из хранилища
	filesInStorage := fileStorage.GetFiles()
	if len(filesInStorage) == 0 {
		return nil, errors.New("no files available for search")
	}

	// Проверяем индекс (O(1))
	if result := checkIndexForKeyword(keyword, fileStorage); result != nil {
		return result, nil
	}

	// Параллельный поиск по файлам
	result, err := parallelSearch(filesInStorage, keyword)
	if err != nil {
		return nil, err
	}

	// Если слово найдено, обновляем индекс
	if len(result) > 0 {
		updateIndexForKeyword(keyword, result, fileStorage)
	}

	return result, nil
}

// checkIndexForKeyword проверяет наличие слова в индексе.
func checkIndexForKeyword(keyword string, fileStorage *storage.FileStorage) []string {
	fileStorage.IdxMu.RLock()
	defer fileStorage.IdxMu.RUnlock()

	if files, found := fileStorage.Index[keyword]; found && len(files) > 0 {
		return files
	}
	return nil
}

// parallelSearch выполняет параллельный поиск слова в списке файлов.
func parallelSearch(files []string, keyword string) ([]string, error) {
	resultChan := make(chan string, len(files))
	errorChan := make(chan error, len(files))
	var _errors []error

	var wg sync.WaitGroup
	for _, filePath := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			found, err := containsWordInFile(path, keyword)
			if err != nil {
				errorChan <- fmt.Errorf("error in file %s: %w", path, err)
				return
			}
			if found {
				resultChan <- path
			}
		}(filePath)
	}

	// Закрываем каналы после завершения всех горутин
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Сбор результатов
	var result []string
	for path := range resultChan {
		result = append(result, path)
	}

	// Проверка на ошибки
	for err := range errorChan {
		_errors = append(_errors, err)
	}

	if len(_errors) > 0 {
		return nil, fmt.Errorf("_errors occurred: %v", _errors)
	}

	return result, nil
}

// containsWordInFile проверяет наличие слова в одном файле.
func containsWordInFile(filePath, keyword string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("warning: failed to open %s to search in: %v", filePath, err)
		return false, fmt.Errorf("failed to open %s to search in: %v", filePath, err)
	}
	defer utils.HandleDeferClose("file to search", file.Close)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), keyword) {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// updateIndexForKeyword добавляет найденные файлы в индекс.
func updateIndexForKeyword(keyword string, files []string, fileStorage *storage.FileStorage) {
	fileStorage.IdxMu.Lock()
	defer fileStorage.IdxMu.Unlock()
	fileStorage.Index[keyword] = files
}
