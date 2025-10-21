package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// addScriptNode is a helper function to inject a script tag into an HTML node.
func addScriptNode(n *html.Node, src string) {
	scriptNode := &html.Node{
		Type: html.ElementNode,
		Data: "script",
		Attr: []html.Attribute{
			{Key: "src", Val: src},
			{Key: "defer", Val: ""},
		},
	}
	n.AppendChild(scriptNode)
}

// htmlHandler serves the main page at /app and other static files.
func htmlHandler(rootDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Serve content under the /app prefix
		if strings.HasPrefix(r.URL.Path, "/app") {
			// Trim the /app prefix to get the relative path
			relativePath := strings.TrimPrefix(r.URL.Path, "/app")
			requestedPath := filepath.Join(rootDir, relativePath)

			info, err := os.Stat(requestedPath)
			if err != nil {
				if os.IsNotExist(err) {
					http.NotFound(w, r)
				} else {
					http.Error(w, "Server error", http.StatusInternalServerError)
					log.Printf("Error stating file %s: %v", requestedPath, err)
				}
				return
			}

			// If it's a directory, look for index.html
			if info.IsDir() {
				if !strings.HasSuffix(r.URL.Path, "/") {
					http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
					return
				}
				requestedPath = filepath.Join(requestedPath, "index.html")
			}

			// Check if the file is an HTML file
			if strings.HasSuffix(strings.ToLower(requestedPath), ".html") {
				content, err := os.ReadFile(requestedPath)
				if err != nil {
					http.NotFound(w, r)
					log.Printf("File not found (or index.html missing): %s", requestedPath)
					return
				}

				doc, err := html.Parse(strings.NewReader(string(content)))
				if err != nil {
					http.Error(w, "Could not parse HTML", http.StatusInternalServerError)
					log.Printf("Error parsing HTML from %s: %v", requestedPath, err)
					return
				}

				// Find head tag and inject script node
				var findHeadAndInject func(*html.Node)
				findHeadAndInject = func(n *html.Node) {
					if n.Type == html.ElementNode && n.Data == "head" {
						addScriptNode(n, "/embed/gohta.js")
						if IsDev {
							addScriptNode(n, "/embed/development.js")
						}
					}

					for c := n.FirstChild; c != nil; c = c.NextSibling {
						findHeadAndInject(c)
					}
				}
				findHeadAndInject(doc)

				// Process image files: convert file:// paths to server URLs and embed relative paths as Base64
				processImageTags(doc, rootDir)

				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				html.Render(w, doc)
				return
			}

			// Serve non-HTML files as static assets
			http.ServeFile(w, r, requestedPath)
			return
		}

		// Redirect root to /app
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/app", http.StatusFound)
			return
		}

		// For any other path, return a 404
		http.NotFound(w, r)
	}
}

// processImageTags traverses HTML nodes and processes src attributes of img tags.
// file:// paths are changed to URLs served by the server,
// and relative path images are converted to Base64 data URIs and embedded in HTML.
func processImageTags(n *html.Node, baseDir string) {
	if n.Type == html.ElementNode && n.Data == "img" {
		for i, attr := range n.Attr {
			if attr.Key == "src" {
				src := attr.Val
				// Handle local file paths starting with file:// protocol
				if strings.HasPrefix(src, "file://") {
					n.Attr[i].Val = convertFileSrc(src)
				} else if !strings.HasPrefix(src, "data:") && !strings.HasPrefix(src, "http") {
					// Embed relative path images by encoding them as Base64
					imagePath := filepath.Join(baseDir, src)
					imageData, err := os.ReadFile(imagePath)
					if err != nil {
						log.Printf("⚠️  Could not read image file for embedding %s: %v", imagePath, err)
						continue // Skip to next attribute if file cannot be read
					}

					// Detect file MIME type
					mimeType := http.DetectContentType(imageData)
					// Create data URI by encoding as Base64
					encoded := base64.StdEncoding.EncodeToString(imageData)
					n.Attr[i].Val = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
				}
				break // Exit loop since src attribute was found
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		processImageTags(c, baseDir)
	}
}

// fileHandler serves files from the local file system through URLs with /file/ prefix.
// Example: /file/C:/Users/user/image.png
func fileHandler(w http.ResponseWriter, r *http.Request) {
	// Remove /file/ prefix from URL path to get actual file path
	filePath := strings.TrimPrefix(r.URL.Path, "/file/")

	// Decode URL-encoded characters (e.g., %20 -> ' ')
	decodedPath, err := url.PathUnescape(filePath)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		log.Printf("❌ Error decoding file path: %v", err)
		return
	}

	// http.ServeFile sanitizes paths for security and finds and serves files from the file system.
	// Care must be taken with security when serving absolute paths.
	http.ServeFile(w, r, decodedPath)
}
