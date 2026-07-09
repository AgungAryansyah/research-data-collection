package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"research-data-collection/internal/config"
	"research-data-collection/internal/storage"
)

var mimeTypes = map[string]string{
	".webm": "video/webm",
	".json": "application/json",
}

func AdminSessionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessions, err := storage.ListSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		sessions = []storage.SessionMeta{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func AdminFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	name := r.URL.Query().Get("file")
	if id == "" || name == "" {
		http.Error(w, "missing id or file", http.StatusBadRequest)
		return
	}

	dir := storage.GetSessionDir(id)
	path := filepath.Clean(filepath.Join(dir, filepath.Base(name)))

	ext := filepath.Ext(name)
	if ct, ok := mimeTypes[ext]; ok {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(name)))
	http.ServeFile(w, r, path)
}

func AdminZipHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	dir := storage.GetSessionDir(id)

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, id))

	zw := zip.NewWriter(w)
	defer zw.Close()

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		w, err := zw.Create(info.Name())
		if err != nil {
			return err
		}
		_, err = io.Copy(w, f)
		return err
	}); err != nil {
		return
	}
}

func AdminConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.Get())

	case http.MethodPost:
		var c config.Config
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if c.AdminUser == "" {
			c.AdminUser = config.Get().AdminUser
		}
		if c.AdminPass == "" {
			c.AdminPass = config.Get().AdminPass
		}
		config.Set(c)
		if err := config.Save("config.json"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.Get())

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func AdminUsageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	usage, err := storage.TotalUsage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"bytes": usage})
}
