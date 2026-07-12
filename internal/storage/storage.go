package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SessionMeta struct {
	UUID      string          `json:"uuid"`
	Name      string          `json:"name"`
	CreatedAt time.Time       `json:"createdAt"`
	Takes     []TakeMeta      `json:"takes"`
	Info      json.RawMessage `json:"info,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type TakeMeta struct {
	Number       int    `json:"number"`
	File         string `json:"file"`
	Completed    bool   `json:"completed"`
	BytesWritten int64  `json:"bytesWritten"`
	StartedAt    string `json:"startedAt"`
}

type SessionWriter struct {
	uuid         string
	take         int
	file         *os.File
	buf          *bufio.Writer
	bytesWritten int64
	startedAt    time.Time
}

var basePath string

func Init(path string) error {
	basePath = path
	return os.MkdirAll(path, 0755)
}

func SessionDir(uuid string) string {
	return filepath.Join(basePath, uuid)
}

func CreateSession(uuid string, info, metadata interface{}) error {
	dir := SessionDir(uuid)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, "info.json"), info); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "metadata.json"), metadata)
}

func NewSessionWriter(uuid string, take int) (*SessionWriter, error) {
	dir := SessionDir(uuid)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	filename := takeFilename(take)
	f, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return nil, err
	}
	return &SessionWriter{
		uuid:      uuid,
		take:      take,
		file:      f,
		buf:       bufio.NewWriterSize(f, 1<<20),
		startedAt: time.Now().UTC(),
	}, nil
}

func (sw *SessionWriter) Write(data []byte) (int, error) {
	n, err := sw.buf.Write(data)
	sw.bytesWritten += int64(n)
	return n, err
}

func (sw *SessionWriter) Close(completed bool) error {
	if err := sw.buf.Flush(); err != nil {
		sw.file.Close()
		return err
	}
	if err := sw.file.Close(); err != nil {
		return err
	}
	return writeTakeMeta(sw.uuid, TakeMeta{
		Number:       sw.take,
		File:         takeFilename(sw.take),
		Completed:    completed,
		BytesWritten: sw.bytesWritten,
		StartedAt:    sw.startedAt.Format(time.RFC3339),
	})
}

func ListSessions() ([]SessionMeta, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}
	var sessions []SessionMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(basePath, e.Name())

		sm := SessionMeta{UUID: e.Name()}

		infoData, err := os.ReadFile(filepath.Join(dir, "info.json"))
		if err == nil {
			sm.Info = infoData
			var infoMap map[string]interface{}
			if json.Unmarshal(infoData, &infoMap) == nil {
				for _, v := range infoMap {
					if s, ok := v.(string); ok {
						sm.Name = s
						break
					}
				}
			}
		}
		metaData, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
		if err == nil {
			sm.Metadata = metaData
			var md struct {
				CollectedAt string `json:"collectedAt"`
			}
			if json.Unmarshal(metaData, &md) == nil && md.CollectedAt != "" {
				sm.CreatedAt, _ = time.Parse(time.RFC3339, md.CollectedAt)
			}
		}

		sm.Takes = readTakeMeta(dir)
		sessions = append(sessions, sm)
	}
	return sessions, nil
}

func GetSessionDir(uuid string) string {
	return SessionDir(uuid)
}

func TotalUsage() (int64, error) {
	var total int64
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func takeFilename(take int) string {
	return fmt.Sprintf("video_take%d.webm", take)
}

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeTakeMeta(uuid string, take TakeMeta) error {
	dir := SessionDir(uuid)
	takes := readTakeMeta(dir)
	for i, t := range takes {
		if t.Number == take.Number {
			takes[i] = take
			return writeJSON(filepath.Join(dir, "takes.json"), takes)
		}
	}
	takes = append(takes, take)
	return writeJSON(filepath.Join(dir, "takes.json"), takes)
}

func readTakeMeta(dir string) []TakeMeta {
	data, err := os.ReadFile(filepath.Join(dir, "takes.json"))
	if err != nil {
		return nil
	}
	var takes []TakeMeta
	json.Unmarshal(data, &takes)
	return takes
}
