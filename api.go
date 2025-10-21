package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// API handler
func apiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Handle POST requests to /api/log path specifically
	if r.URL.Path == "/api/log" && r.Method == http.MethodPost {
		handleLogRequest(w, r)
		return
	}

	// Handle POST requests to /api/core/convertFileSrc path
	if r.URL.Path == "/api/core/convertFileSrc" && r.Method == http.MethodPost {
		handleConvertFileSrcRequest(w, r)
		return
	}

	// Handle POST requests to /api/core/getArgs path
	if r.URL.Path == "/api/core/getArgs" && r.Method == http.MethodGet {
		handleGetArgsRequest(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		fmt.Fprintf(w, `{"message":"Hello from Go API!","method":"GET","path":"%s"}`, r.URL.Path)
	case http.MethodPost:
		// Other POST requests (not /api/log) can be handled here
		fmt.Fprintf(w, `{"message":"Data received","method":"POST"}`)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"error":"Method not allowed","allowed":["GET","POST"]}`)
	}
}

// handleLogRequest processes POST requests to /api/log.
func handleLogRequest(w http.ResponseWriter, r *http.Request) {
	payload := struct {
		Message string `json:"message"`
	}{}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("❌  Error decoding log request: %v", err)
		return
	}

	// Output log to server console
	log.Printf("💻 [CLIENT] %s", payload.Message)

	// Send success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleConvertFileSrcRequest processes POST requests to /api/convertFileSrc.
func handleConvertFileSrcRequest(w http.ResponseWriter, r *http.Request) {
	payload := struct {
		FilePath string `json:"filePath"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		log.Printf("❌  Error decoding convertFileSrc request: %v", err)
		return
	}

	// Remove file:// prefix and add /file/ prefix to create new source URL.
	newSrc := convertFileSrc(payload.FilePath)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newSrc)
}

// handleGetArgsRequest returns the command-line arguments passed to the application.
func handleGetArgsRequest(w http.ResponseWriter, r *http.Request) {
	var args []string
	if staticMode {
		// In static mode, all arguments after the executable are returned
		args = os.Args[1:]
	} else {
		// In local mode, the first argument is the file path, so we return the rest
		if len(os.Args) > 2 {
			args = os.Args[2:]
		} else {
			args = []string{}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(args)
}
