// cmd/rest-server/main.go
package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Upload endpoint (not implemented yet)\n"))
	})

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Download endpoint (not implemented yet)\n"))
	})

	log.Println("REST server listening on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
