//go:build dev

package main

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const IsDev = true

// WebSocket upgrader for live reload
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from localhost
		return true
	},
}

// WebSocket connection manager
type ConnectionManager struct {
	connections map[*websocket.Conn]bool
	mutex       sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[*websocket.Conn]bool),
	}
}

// AddConnection adds a new WebSocket connection
func (cm *ConnectionManager) AddConnection(conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.connections[conn] = true
	log.Printf("🔌 WebSocket connection added. Total connections: %d", len(cm.connections))
}

// RemoveConnection removes a WebSocket connection
func (cm *ConnectionManager) RemoveConnection(conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.connections, conn)
	log.Printf("🔌 WebSocket connection removed. Total connections: %d", len(cm.connections))
}

// BroadcastMessage sends a message to all connected clients
func (cm *ConnectionManager) BroadcastMessage(message string) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for conn := range cm.connections {
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("❌ Error sending WebSocket message: %v", err)
			conn.Close()
			delete(cm.connections, conn)
		}
	}
}

// Global connection manager
var connManager = NewConnectionManager()

// WebSocket handler for live reload
func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	connManager.AddConnection(conn)
	defer connManager.RemoveConnection(conn)

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error: %v", err)
			}
			break
		}
	}
}

// File watcher for live reload
func startFileWatcher(watchDir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("❌ Error creating file watcher: %v", err)
		return
	}
	defer watcher.Close()

	// Add watch directory
	err = watcher.Add(watchDir)
	if err != nil {
		log.Printf("❌ Error adding watch directory: %v", err)
		return
	}

	log.Printf("👀 File watcher started for directory: %s", watchDir)

	// Debounce timer to prevent multiple rapid reloads
	var debounceTimer *time.Timer
	var debounceMutex sync.Mutex

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only watch for write events on relevant files
			if event.Op&fsnotify.Write == fsnotify.Write {
				ext := strings.ToLower(filepath.Ext(event.Name))
				// Watch for HTML, CSS, JS, and other web files
				if ext == ".html" || ext == ".css" || ext == ".js" || ext == ".json" || ext == ".xml" {
					log.Printf("📝 File changed: %s", event.Name)

					// Debounce rapid file changes
					debounceMutex.Lock()
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
						log.Printf("🔄 Sending reload signal to %d connected clients", len(connManager.connections))
						connManager.BroadcastMessage("reload")
					})
					debounceMutex.Unlock()
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("❌ File watcher error: %v", err)
		}
	}
}

// Development mode initialization
func initDevMode(mux *http.ServeMux, htmlFileDir string) {
	log.Println("🚀 Development mode enabled")

	// Register WebSocket handler
	mux.HandleFunc("/ws", websocketHandler)

	// Start file watcher in a goroutine
	go startFileWatcher(htmlFileDir)
}
