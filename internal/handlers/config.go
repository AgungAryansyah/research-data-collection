package handlers

import (
	"encoding/json"
	"net/http"

	"research-data-collection/internal/config"
)

type publicConfig struct {
	ChunkDurationMs int      `json:"chunkDurationMs"`
	VideoBitrate    int      `json:"videoBitrate"`
	MaxWidth        int      `json:"maxWidth"`
	MaxHeight       int      `json:"maxHeight"`
	InfoFields      []string `json:"infoFields"`
	AudioEnabled    bool     `json:"audioEnabled"`
}

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	c := config.Get()
	pub := publicConfig{
		ChunkDurationMs: c.ChunkDurationMs,
		VideoBitrate:    c.VideoBitrate,
		MaxWidth:        c.MaxWidth,
		MaxHeight:       c.MaxHeight,
		InfoFields:      c.InfoFields,
		AudioEnabled:    c.AudioEnabled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pub)
}
