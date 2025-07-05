package metastore

// ChunkInfo содержит информацию о чанке
type ChunkInfo struct {
	ChunkID string // SHA-256 хеш содержимого чанка
	NodeURL string // URL сервера хранения, где находится чанк
}

// FileMeta содержит метаданные о файле
type FileMeta struct {
	FileID      string              // Уникальный идентификатор файла
	Filename    string              // Имя файла
	TotalChunks int                 // Общее количество частей
	Chunks      map[int][]ChunkInfo // Карта индексов частей к информации о частях (с репликами)
	Complete    bool                // Флаг завершенности загрузки
}

// MetaStore интерфейс для хранения метаданных
type MetaStore interface {
	// InitFile инициализирует новую запись о файле
	InitFile(fileID, filename string, size int64) error

	// SaveChunk сохраняет информацию о чанке файла
	SaveChunk(fileID string, index int, info ChunkInfo) error

	// SaveChunkHash сохраняет информацию о чанке по его хешу
	SaveChunkHash(hash string, info ChunkInfo) error

	// HasChunkByHash проверяет наличие чанка с указанным хешем
	HasChunkByHash(hash string) (bool, ChunkInfo, error)

	// GetFileMeta возвращает метаданные о файле
	GetFileMeta(fileID string) (*FileMeta, error)

	// MarkComplete помечает файл как полностью загруженный
	MarkComplete(fileID string) error

	// Close закрывает хранилище
	Close() error
}
