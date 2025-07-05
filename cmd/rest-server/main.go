// cmd/rest-server/main.go
package main

import (
	"flag"
	"github.com/Gammanik/distributed-storage/internal/api"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Gammanik/distributed-storage/internal/metastore"
	"github.com/Gammanik/distributed-storage/internal/storage"
)

var (
	port        = flag.Int("port", 8080, "HTTP port to listen on")
	metaDBPath  = flag.String("meta", "/data/meta.db", "Path to metadata database")
	chunkSize   = flag.Int64("chunk-size", 64<<20, "Default chunk size in bytes")
	storagePool = flag.String("storage-pool", "http://storage1:9000,http://storage2:9000", "Comma-separated list of storage nodes")
)

func main() {
	flag.Parse()

	// Инициализируем хранилище метаданных
	store, err := metastore.NewBoltStore(*metaDBPath)
	if err != nil {
		log.Fatalf("Failed to open metastore: %v", err)
	}
	defer store.Close()

	// Разбиваем список серверов хранения
	nodes := strings.Split(*storagePool, ",")
	for i, node := range nodes {
		nodes[i] = strings.TrimSpace(node)
	}

	// Создаем обработчик файлов
	fileHandler := &api.FileHandler{
		Store:       store,
		Storage:     storage.New(),
		StoragePool: nodes,
		ChunkSize:   *chunkSize,
	}

	// Регистрируем обработчики HTTP запросов
	http.HandleFunc("/upload", fileHandler.Upload)
	http.HandleFunc("/download", fileHandler.Download)
	http.HandleFunc("/info", fileHandler.GetFileInfo)

	// Настраиваем и запускаем HTTP сервер
	server := &http.Server{
		Addr:         ":" + strconv.Itoa(*port),
		Handler:      http.DefaultServeMux,
		ReadTimeout:  300 * time.Second,
		WriteTimeout: 300 * time.Second,
	}

	log.Printf("REST server starting on :%d", *port)
	log.Printf("Connected to %d storage nodes", len(nodes))
	log.Fatal(server.ListenAndServe())
}
