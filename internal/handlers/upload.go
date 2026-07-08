package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"

	"research-data-collection/internal/storage"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type metaMessage struct {
	Info     interface{} `json:"info"`
	Metadata interface{} `json:"metadata"`
}

type finalizeMessage struct {
	Type string `json:"type"`
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	sessionUUID := r.URL.Query().Get("session")
	if sessionUUID == "" {
		http.Error(w, "missing session", http.StatusBadRequest)
		return
	}

	takeStr := r.URL.Query().Get("take")
	takeNum, err := strconv.Atoi(takeStr)
	if err != nil || takeNum < 1 {
		http.Error(w, "invalid take number", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}
	defer conn.Close()
	conn.SetReadLimit(100 << 20)

	_, metaBytes, err := conn.ReadMessage()
	if err != nil {
		log.Printf("ws read meta: %v", err)
		return
	}

	var meta metaMessage
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		log.Printf("ws bad meta: %v", err)
		return
	}

	if err := storage.CreateSession(sessionUUID, meta.Info, meta.Metadata); err != nil {
		log.Printf("ws create session: %v", err)
		return
	}

	sw, err := storage.NewSessionWriter(sessionUUID, takeNum)
	if err != nil {
		log.Printf("ws open take: %v", err)
		return
	}

	var finalized bool
	defer func() { sw.Close(finalized) }()
	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		switch mt {
		case websocket.BinaryMessage:
			if _, err := sw.Write(msg); err != nil {
				log.Printf("ws write chunk: %v", err)
				return
			}
		case websocket.TextMessage:
			var fm finalizeMessage
			if json.Unmarshal(msg, &fm) == nil && fm.Type == "finalize" {
				finalized = true
				return
			}
		}
	}
}
