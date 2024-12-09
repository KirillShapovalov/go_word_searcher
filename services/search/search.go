package search

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/KirillShapovalov/go_word_searcher/services/fileUtils"
	"github.com/KirillShapovalov/go_word_searcher/storage"
)

func FindWordInFiles(keyword string, fileStorage *storage.FileStorage) ([]string, error) {
	// Получаем файлы из хранилища
	filesInStorage := fileStorage.FileManager.GetFiles()
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
	fileStorage.IndexManager.Mu.RLock()
	defer fileStorage.IndexManager.Mu.RUnlock()

	if files, found := fileStorage.IndexManager.Index[keyword]; found && len(files) > 0 {
		return files
	}
	return nil
}

// parallelSearch выполняет параллельный поиск слова в списке файлов.
func parallelSearch(files []string, keyword string) ([]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultChan := make(chan string, len(files))
	errorChan := make(chan error, len(files))
	var _errors []error

	var wg sync.WaitGroup
	for _, filePath := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			found, err := containsWordInFile(ctx, path, keyword)
			if err != nil {
				select {
				case errorChan <- fmt.Errorf("error in file %s: %w", path, err):
					cancel()
				case <-ctx.Done():
				}
				return
			}
			if found {
				select {
				case resultChan <- path:
					cancel()
				case <-ctx.Done():
				}
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
	for {
		select {
		case path, ok := <-resultChan:
			if !ok {
				resultChan = nil
			} else {
				result = append(result, path)
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else {
				_errors = append(_errors, err)
			}
		}
		if resultChan == nil && errorChan == nil {
			break
		}
	}

	if len(_errors) > 0 {
		return nil, fmt.Errorf("_errors occurred: %v", _errors)
	}

	return result, nil
}

// containsWordInFile проверяет наличие слова в одном файле.
func containsWordInFile(ctx context.Context, filePath, keyword string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("warning: failed to open %s to search in: %v", filePath, err)
		return false, fmt.Errorf("failed to open %s to search in: %v", filePath, err)
	}
	defer fileUtils.HandleDeferClose("file to search", file.Close)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		if strings.Contains(scanner.Text(), keyword) {
			return true, nil
		}
	}

	return false, scanner.Err()
}

// updateIndexForKeyword добавляет найденные файлы в индекс.
func updateIndexForKeyword(keyword string, files []string, fileStorage *storage.FileStorage) {
	fileStorage.IndexManager.Mu.Lock()
	defer fileStorage.IndexManager.Mu.Unlock()
	fileStorage.IndexManager.Index[keyword] = files
}
