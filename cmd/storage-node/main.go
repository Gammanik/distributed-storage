package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const storageDir = "/tmp/chunks"

func main() {
	err := os.MkdirAll(storageDir, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create storage dir: %v", err)
	}

	http.HandleFunc("/store/", handleStore)
	http.HandleFunc("/get/", handleGet)

	log.Println("Storage node listening on :9000")
	err = http.ListenAndServe(":9000", nil)
	if err != nil {
		log.Fatalf("Storage node failed: %v", err)
	}
}

func handleStore(w http.ResponseWriter, r *http.Request) {
	chunkID := strings.TrimPrefix(r.URL.Path, "/store/")
	path := storageDir + "/" + chunkID

	file, err := os.Create(path)
	if err != nil {
		http.Error(w, "failed to create file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "failed to write chunk", http.StatusInternalServerError)
		return
	}

	log.Printf("Stored chunk %s\n", chunkID)
	w.WriteHeader(http.StatusCreated)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	chunkID := strings.TrimPrefix(r.URL.Path, "/get/")
	path := storageDir + "/" + chunkID

	file, err := os.Open(path)
	if err != nil {
		http.Error(w, "chunk not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "failed to stream chunk", http.StatusInternalServerError)
		return
	}

	log.Printf("Served chunk %s\n", chunkID)
}
