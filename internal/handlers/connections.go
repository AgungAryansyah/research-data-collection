package handlers

import (
	"sort"
	"strconv"
	"sync"
	"time"
)

type ConnMeta struct {
	SessionUUID string    `json:"sessionUUID"`
	Take        int       `json:"take"`
	ClientIP    string    `json:"clientIP"`
	UserAgent   string    `json:"userAgent"`
	ConnectedAt time.Time `json:"connectedAt"`
	BytesSent   int64     `json:"bytesSent"`
}

var (
	conns   = make(map[string]*ConnMeta)
	connsMu sync.Mutex
)

func connKey(uuid string, take int) string {
	return uuid + "-" + strconv.Itoa(take)
}

func RegisterConn(uuid, clientIP, userAgent string, take int) *ConnMeta {
	cm := &ConnMeta{
		SessionUUID: uuid,
		Take:        take,
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		ConnectedAt: time.Now().UTC(),
	}
	connsMu.Lock()
	conns[connKey(uuid, take)] = cm
	connsMu.Unlock()
	return cm
}

func UnregisterConn(uuid string, take int) {
	connsMu.Lock()
	delete(conns, connKey(uuid, take))
	connsMu.Unlock()
}

func ListConns() []ConnMeta {
	connsMu.Lock()
	defer connsMu.Unlock()
	result := make([]ConnMeta, 0, len(conns))
	for _, cm := range conns {
		result = append(result, *cm)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ConnectedAt.Before(result[j].ConnectedAt)
	})
	return result
}


