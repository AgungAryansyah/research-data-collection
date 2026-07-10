package main

import (
	"log"
	"net/http"
	"os"
	"time"

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

	http.HandleFunc("/api/admin/config", handlers.BasicAuth(handlers.AdminConfigHandler))
	http.HandleFunc("/api/admin/sessions", handlers.BasicAuth(handlers.AdminSessionsHandler))
	http.HandleFunc("/api/admin/sessions/file", handlers.BasicAuth(handlers.AdminFileHandler))
	http.HandleFunc("/api/admin/sessions/zip", handlers.BasicAuth(handlers.AdminZipHandler))
	http.HandleFunc("/api/admin/usage", handlers.BasicAuth(handlers.AdminUsageHandler))
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/dashboard.html")
	})

	addr := os.Getenv("BIND_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:              addr,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}
