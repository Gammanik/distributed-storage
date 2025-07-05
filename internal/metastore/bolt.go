// internal/metastore/bolt.go
package metastore

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	filesBucket  = []byte("files")
	chunksBucket = []byte("chunks")
)

// BoltStore реализация MetaStore на основе BoltDB
type BoltStore struct {
	db *bolt.DB
}

// NewBoltStore создает новое хранилище метаданных на основе BoltDB
func NewBoltStore(path string) (*BoltStore, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	// Создаем необходимые бакеты
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(filesBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(chunksBucket)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &BoltStore{db: db}, nil
}

// InitFile инициализирует новую запись о файле
func (bs *BoltStore) InitFile(fileID, filename string, size int64) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(filesBucket)
		meta := FileMeta{
			FileID:      fileID,
			Filename:    filename,
			TotalChunks: 0,
			Chunks:      make(map[int][]ChunkInfo),
			Complete:    false,
		}

		encoded, err := json.Marshal(meta)
		if err != nil {
			return err
		}

		return b.Put([]byte(fileID), encoded)
	})
}

// SaveChunk сохраняет информацию о чанке файла
func (bs *BoltStore) SaveChunk(fileID string, index int, info ChunkInfo) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(filesBucket)

		data := b.Get([]byte(fileID))
		if data == nil {
			return fmt.Errorf("file not found: %s", fileID)
		}

		var meta FileMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return err
		}

		// Добавляем или обновляем информацию о чанке
		if _, exists := meta.Chunks[index]; !exists {
			meta.Chunks[index] = []ChunkInfo{info}
		} else {
			// Проверяем, нет ли уже такого узла
			exists := false
			for _, ci := range meta.Chunks[index] {
				if ci.NodeURL == info.NodeURL {
					exists = true
					break
				}
			}
			if !exists {
				meta.Chunks[index] = append(meta.Chunks[index], info)
			}
		}

		// Обновляем общее количество чанков, если нужно
		if index+1 > meta.TotalChunks {
			meta.TotalChunks = index + 1
		}

		encoded, err := json.Marshal(meta)
		if err != nil {
			return err
		}

		return b.Put([]byte(fileID), encoded)
	})
}

// SaveChunkHash сохраняет информацию о чанке по его хешу
func (bs *BoltStore) SaveChunkHash(hash string, info ChunkInfo) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(chunksBucket)

		// Кодируем информацию о чанке
		encoded, err := json.Marshal(info)
		if err != nil {
			return err
		}

		return b.Put([]byte(hash), encoded)
	})
}

// HasChunkByHash проверяет наличие чанка с указанным хешем
func (bs *BoltStore) HasChunkByHash(hash string) (bool, ChunkInfo, error) {
	var info ChunkInfo

	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(chunksBucket)
		data := b.Get([]byte(hash))

		if data == nil {
			return nil // Чанк не найден
		}

		return json.Unmarshal(data, &info)
	})

	if err != nil {
		return false, info, err
	}

	return info.ChunkID != "", info, nil
}

// GetFileMeta возвращает метаданные о файле
func (bs *BoltStore) GetFileMeta(fileID string) (*FileMeta, error) {
	var meta FileMeta

	err := bs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(filesBucket)
		data := b.Get([]byte(fileID))

		if data == nil {
			return fmt.Errorf("file not found: %s", fileID)
		}

		return json.Unmarshal(data, &meta)
	})

	if err != nil {
		return nil, err
	}

	return &meta, nil
}

// MarkComplete помечает файл как полностью загруженный
func (bs *BoltStore) MarkComplete(fileID string) error {
	return bs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(filesBucket)

		data := b.Get([]byte(fileID))
		if data == nil {
			return fmt.Errorf("file not found: %s", fileID)
		}

		var meta FileMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return err
		}

		meta.Complete = true

		encoded, err := json.Marshal(meta)
		if err != nil {
			return err
		}

		return b.Put([]byte(fileID), encoded)
	})
}

// Close закрывает хранилище
func (bs *BoltStore) Close() error {
	return bs.db.Close()
}
