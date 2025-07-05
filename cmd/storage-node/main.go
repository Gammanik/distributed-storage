// cmd/storage-node/main.go
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"syscall"

	"github.com/gorilla/mux"
)

var (
	port    = flag.Int("port", 9000, "HTTP port to listen on")
	dataDir = flag.String("data", "./data", "Directory to store chunks")
	nodeID  = flag.String("id", "", "Node ID (default: from environment NODE_ID)")
)

func main() {
	flag.Parse()

	// Используем ID из аргумента или переменной окружения
	id := *nodeID
	if id == "" {
		id = os.Getenv("NODE_ID")
		if id == "" {
			log.Fatal("Node ID is required. Set NODE_ID environment variable or use -id flag.")
		}
	}

	// Создаем директорию для хранения данных
	storageDir := filepath.Join(*dataDir, id)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	router := mux.NewRouter()

	// Обработчик для загрузки чанка
	router.HandleFunc("/chunks/{chunkID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		vars := mux.Vars(r)
		chunkID := vars["chunkID"]

		// Проверяем, что chunkID является валидным SHA-256 хешем
		if len(chunkID) != 64 || !isHexString(chunkID) {
			http.Error(w, "Invalid chunk ID", http.StatusBadRequest)
			return
		}

		// Создаем путь для сохранения чанка
		chunkPath := filepath.Join(storageDir, chunkID)

		// Если чанк уже существует, ничего не делаем
		if _, err := os.Stat(chunkPath); err == nil {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Chunk already exists")
			return
		}

		// Создаем временный файл
		tmpPath := chunkPath + ".tmp"
		file, err := os.Create(tmpPath)
		if err != nil {
			http.Error(w, "Failed to create file", http.StatusInternalServerError)
			log.Printf("Failed to create file: %v", err)
			return
		}
		defer file.Close()

		// Читаем данные и вычисляем хеш
		h := sha256.New()
		writer := io.MultiWriter(file, h)

		if _, err := io.Copy(writer, r.Body); err != nil {
			http.Error(w, "Failed to read data", http.StatusInternalServerError)
			log.Printf("Failed to read data: %v", err)
			os.Remove(tmpPath)
			return
		}

		// Проверяем, что хеш данных соответствует chunkID
		actualHash := hex.EncodeToString(h.Sum(nil))
		if actualHash != chunkID {
			http.Error(w, "Hash mismatch", http.StatusBadRequest)
			log.Printf("Hash mismatch. Expected: %s, Got: %s", chunkID, actualHash)
			os.Remove(tmpPath)
			return
		}

		// Переименовываем временный файл
		if err := os.Rename(tmpPath, chunkPath); err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			log.Printf("Failed to rename file: %v", err)
			os.Remove(tmpPath)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, "Chunk saved")
	}).Methods("PUT")

	// Обработчик для скачивания чанка
	router.HandleFunc("/chunks/{chunkID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		vars := mux.Vars(r)
		chunkID := vars["chunkID"]

		// Проверяем, что chunkID является валидным SHA-256 хешем
		if len(chunkID) != 64 || !isHexString(chunkID) {
			http.Error(w, "Invalid chunk ID", http.StatusBadRequest)
			return
		}

		// Путь к файлу
		chunkPath := filepath.Join(storageDir, chunkID)

		// Проверяем существование файла
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			http.Error(w, "Chunk not found", http.StatusNotFound)
			return
		}

		// Открываем файл для чтения
		file, err := os.Open(chunkPath)
		if err != nil {
			http.Error(w, "Failed to read chunk", http.StatusInternalServerError)
			log.Printf("Failed to open file: %v", err)
			return
		}
		defer file.Close()

		// Отправляем содержимое файла
		w.Header().Set("Content-Type", "application/octet-stream")
		if _, err := io.Copy(w, file); err != nil {
			log.Printf("Failed to send file: %v", err)
		}
	}).Methods("GET")

	// Обработчик для удаления чанка (для администрирования)
	router.HandleFunc("/chunks/{chunkID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		vars := mux.Vars(r)
		chunkID := vars["chunkID"]

		// Проверяем, что chunkID является валидным SHA-256 хешем
		if len(chunkID) != 64 || !isHexString(chunkID) {
			http.Error(w, "Invalid chunk ID", http.StatusBadRequest)
			return
		}

		// Путь к файлу
		chunkPath := filepath.Join(storageDir, chunkID)

		// Проверяем существование файла
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			http.Error(w, "Chunk not found", http.StatusNotFound)
			return
		}

		// Удаляем файл
		if err := os.Remove(chunkPath); err != nil {
			http.Error(w, "Failed to delete chunk", http.StatusInternalServerError)
			log.Printf("Failed to delete file: %v", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Chunk deleted")
	}).Methods("DELETE")

	// Обработчик статуса узла
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Получаем информацию о свободном месте
		var stat syscall.Statfs_t
		syscall.Statfs(storageDir, &stat)

		// Считаем количество чанков
		chunks, err := countChunks(storageDir)
		if err != nil {
			log.Printf("Failed to count chunks: %v", err)
		}

		// Считаем общий размер данных
		totalSize, err := dirSize(storageDir)
		if err != nil {
			log.Printf("Failed to calculate storage size: %v", err)
		}

		fmt.Fprintf(w, `{
			"nodeID": "%s",
			"status": "online",
			"chunks": %d,
			"totalSize": %d,
			"freeSpace": %d
		}`, id, chunks, totalSize, stat.Bfree*uint64(stat.Bsize))
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Storage node %s starting on %s", id, addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

// isHexString проверяет, что строка содержит только шестнадцатеричные символы
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// countChunks подсчитывает количество чанков в директории
func countChunks(dir string) (int, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, file := range files {
		if !file.IsDir() && len(file.Name()) == 64 && isHexString(file.Name()) {
			count++
		}
	}

	return count, nil
}

// dirSize вычисляет общий размер директории
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
