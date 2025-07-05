package api

import (
	"encoding/json"
	"fmt"
	"github.com/Gammanik/distributed-storage/internal/metastore"
	"github.com/Gammanik/distributed-storage/internal/storage"
	"github.com/Gammanik/distributed-storage/internal/utils"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

// FileHandler обрабатывает запросы для файлов
type FileHandler struct {
	Store       metastore.MetaStore
	Storage     storage.Client
	StoragePool []string
	ChunkSize   int64
}

// Upload обрабатывает загрузку файла
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Генерируем уникальный ID для файла
	fileID := uuid.NewString()

	// Получаем имя файла из заголовка
	filename := r.Header.Get("X-Filename")
	if filename == "" {
		filename = "uploaded.bin"
	}

	// Получаем размер чанка из заголовка или используем значение по умолчанию
	chunkSize := h.ChunkSize
	if v := r.Header.Get("X-Chunk-Size"); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 {
			chunkSize = int64(s)
		}
	}

	// Инициализируем запись о файле в метаданных
	if err := h.Store.InitFile(fileID, filename, 0); err != nil {
		http.Error(w, "failed to init file", http.StatusInternalServerError)
		log.Printf("Failed to init file: %v", err)
		return
	}

	// Создаем reader для чтения чанков
	chunkReader := chunker.NewChunkReader(r.Body, chunkSize)
	index := 0

	// Читаем и загружаем чанки
	for {
		chunk, hash, err := chunkReader.NextChunk()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "reading failed", http.StatusInternalServerError)
			log.Printf("Failed to read chunk: %v", err)
			return
		}

		// Проверяем, существует ли уже чанк с таким хешем
		if found, ci, _ := h.Store.HasChunkByHash(hash); found {
			// Чанк уже существует, просто сохраняем его ссылку
			h.Store.SaveChunk(fileID, index, ci)
			index++
			continue
		}

		// Выбираем серверы для хранения чанка
		nodes := utils.ChooseStorageNodes(index, h.StoragePool, 2)
		primaryNode := nodes[0]

		// Загружаем чанк на основной сервер
		if err := h.Storage.UploadChunk(hash, primaryNode, chunk); err != nil {
			http.Error(w, "upload failed", http.StatusBadGateway)
			log.Printf("Failed to upload chunk to primary node: %v", err)
			return
		}

		// Создаем информацию о чанке
		ci := metastore.ChunkInfo{ChunkID: hash, NodeURL: primaryNode}

		// Сохраняем информацию о чанке
		h.Store.SaveChunk(fileID, index, ci)
		h.Store.SaveChunkHash(hash, ci)

		// Загружаем чанк на резервные серверы (best effort)
		for i := 1; i < len(nodes); i++ {
			if err := h.Storage.UploadChunk(hash, nodes[i], chunk); err != nil {
				log.Printf("Warning: failed to upload replica to %s: %v", nodes[i], err)
				continue
			}

			// Сохраняем информацию о реплике
			replica := metastore.ChunkInfo{ChunkID: hash, NodeURL: nodes[i]}
			h.Store.SaveChunk(fileID, index, replica)
		}

		index++
	}

	// Помечаем файл как полностью загруженный
	h.Store.MarkComplete(fileID)

	// Отвечаем клиенту идентификатором файла
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, fileID)
}

// Download обрабатывает скачивание файла
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	// Получаем ID файла из query параметра
	fileID := r.URL.Query().Get("fileID")
	if fileID == "" {
		http.Error(w, "missing fileID", http.StatusBadRequest)
		return
	}

	// Получаем метаданные о файле
	meta, err := h.Store.GetFileMeta(fileID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		log.Printf("File not found: %v", err)
		return
	}

	// Проверяем, что файл полностью загружен
	if !meta.Complete {
		http.Error(w, "file is not fully uploaded", http.StatusBadRequest)
		return
	}

	// Устанавливаем заголовки для скачивания
	w.Header().Set("Content-Disposition", "attachment; filename=\""+meta.Filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")

	// Скачиваем и отправляем каждый чанк
	for i := 0; i < meta.TotalChunks; i++ {
		replicas := meta.Chunks[i]
		var success bool

		// Пробуем скачать чанк с одного из доступных серверов
		for _, replica := range replicas {
			data, err := h.Storage.DownloadChunk(replica.ChunkID, replica.NodeURL)
			if err == nil {
				// Проверяем целостность данных
				hash := utils.CalculateSHA256(data)
				if hash != replica.ChunkID {
					log.Printf("Warning: chunk hash mismatch. Expected: %s, Got: %s", replica.ChunkID, hash)
					continue
				}

				// Отправляем чанк клиенту
				if _, err := w.Write(data); err != nil {
					log.Printf("Failed to write chunk to response: %v", err)
					http.Error(w, "download failed", http.StatusInternalServerError)
					return
				}

				success = true
				break
			}

			log.Printf("Failed to download chunk from %s: %v", replica.NodeURL, err)
		}

		if !success {
			http.Error(w, "missing chunk", http.StatusInternalServerError)
			log.Printf("All replicas for chunk %d of file %s are unavailable", i, fileID)
			return
		}
	}
}

// GetFileInfo возвращает информацию о файле
func (h *FileHandler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("fileID")
	if fileID == "" {
		http.Error(w, "missing fileID", http.StatusBadRequest)
		return
	}

	meta, err := h.Store.GetFileMeta(fileID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fileID":      meta.FileID,
		"filename":    meta.Filename,
		"totalChunks": meta.TotalChunks,
		"complete":    meta.Complete,
	})
}
