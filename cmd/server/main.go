package main

import (
	"log"
	"net/http"

	"research-data-collection/internal/config"
	"research-data-collection/internal/handlers"
	"research-data-collection/internal/storage"
)

func main() {
	if err := config.Load("config.json"); err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := storage.Init(config.Get().StoragePath); err != nil {
		log.Fatalf("storage: %v", err)
	}

	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", fs)
	http.HandleFunc("/api/config", handlers.ConfigHandler)
	http.HandleFunc("/ws/upload", handlers.UploadHandler)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
