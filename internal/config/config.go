package config

import (
	"encoding/json"
	"os"
	"sync"
)

type Config struct {
	StoragePath     string   `json:"storagePath"`
	ChunkDurationMs int      `json:"chunkDurationMs"`
	VideoBitrate    int      `json:"videoBitrate"`
	MaxWidth        int      `json:"maxWidth"`
	MaxHeight       int      `json:"maxHeight"`
	AdminUser       string   `json:"adminUser"`
	AdminPass       string   `json:"adminPass"`
	InfoFields      []string `json:"infoFields"`
	AudioEnabled    bool     `json:"audioEnabled"`
}

var (
	cfg Config
	mu  sync.RWMutex
)

func Defaults() Config {
	return Config{
		StoragePath:     "uploads",
		ChunkDurationMs: 500,
		VideoBitrate:    5000000,
		MaxWidth:        1920,
		MaxHeight:       1080,
		AdminUser:       "admin",
		AdminPass:       "admin",
		InfoFields:      []string{"Name", "Age", "Notes"},
		AudioEnabled:    false,
	}
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

func Set(c Config) {
	mu.Lock()
	defer mu.Unlock()
	cfg = c
}

func Load(path string) error {
	c := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		mu.Lock()
		cfg = c
		mu.Unlock()
		return Save(path)
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}
	mu.Lock()
	cfg = c
	mu.Unlock()
	return nil
}

func Save(path string) error {
	mu.RLock()
	defer mu.RUnlock()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
