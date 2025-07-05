package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// Client интерфейс для взаимодействия с серверами хранения
type Client interface {
	// UploadChunk загружает чанк на указанный сервер хранения
	UploadChunk(chunkID, nodeURL string, data []byte) error

	// DownloadChunk скачивает чанк с указанного сервера хранения
	DownloadChunk(chunkID, nodeURL string) ([]byte, error)
}

// HTTPClient реализация Client для взаимодействия с серверами хранения через HTTP
type HTTPClient struct {
	client *http.Client
}

// New создает новый HTTP клиент для серверов хранения
func New() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{},
	}
}

// UploadChunk загружает чанк на указанный сервер хранения
func (c *HTTPClient) UploadChunk(chunkID, nodeURL string, data []byte) error {
	url := fmt.Sprintf("%s/chunks/%s", nodeURL, chunkID)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload chunk: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// DownloadChunk скачивает чанк с указанного сервера хранения
func (c *HTTPClient) DownloadChunk(chunkID, nodeURL string) ([]byte, error) {
	url := fmt.Sprintf("%s/chunks/%s", nodeURL, chunkID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download chunk: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
