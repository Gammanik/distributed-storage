package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// CalculateSHA256 вычисляет SHA-256 хеш данных
func CalculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CalculateFileSHA256 вычисляет SHA-256 хеш содержимого файла
func CalculateFileSHA256(reader io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
