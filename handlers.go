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
func addScriptNode(n *html.Node, src string, isDefer bool) {
	scriptNode := &html.Node{
		Type: html.ElementNode,
		Data: "script",
		Attr: []html.Attribute{
			{Key: "src", Val: src},
			ternary(isDefer, html.Attribute{Key: "defer", Val: ""}, html.Attribute{}),
		},
	}
	n.AppendChild(scriptNode)
}

// htmlHandler serves static files and processes HTML files for script injection.
func htmlHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle routes outside of /app
		if !strings.HasPrefix(r.URL.Path, "/app") {
			if r.URL.Path == "/" {
				http.Redirect(w, r, "/app/", http.StatusFound)
			} else {
				http.NotFound(w, r)
			}
			return
		}

		// Get the relative path of the requested file
		relativePath := strings.TrimPrefix(r.URL.Path, "/app")
		relativePath = strings.TrimPrefix(relativePath, "/")

		safePath := relativePath
		if safePath == "" {
			safePath = "."
		}
		// Check if the path is a directory and handle redirects/index.html
		file, err := contentFS.Open(safePath)
		if err == nil {
			info, err := file.Stat()
			if err == nil && info.IsDir() {
				if !strings.HasSuffix(r.URL.Path, "/") {
					http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
					file.Close()
					return
				}
				relativePath = filepath.Join(relativePath, "index.html")
			}
			file.Close()
		}

		// If it's an HTML file, process it
		if strings.HasSuffix(strings.ToLower(relativePath), ".html") {
			content, err := readFile(relativePath)
			if err != nil {
				http.NotFound(w, r)
				log.Printf("File not found: %s", relativePath)
				return
			}

			doc, err := html.Parse(strings.NewReader(string(content)))
			if err != nil {
				http.Error(w, "Could not parse HTML", http.StatusInternalServerError)
				log.Printf("Error parsing HTML from %s: %v", relativePath, err)
				return
			}

			var findHeadAndInject func(*html.Node)
			findHeadAndInject = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "head" {
					addScriptNode(n, "/embed/gohta.js", false)
					if IsDev {
						addScriptNode(n, "/embed/development.js", true)
					}
					return
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					findHeadAndInject(c)
				}
			}
			findHeadAndInject(doc)

			processImageTags(doc)

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			html.Render(w, doc)
			return
		}

		// For everything else, serve it as a static file
		r.URL.Path = relativePath // Temporarily rewrite the path for the static server
		staticServer.ServeHTTP(w, r)
	}
}

func readFile(path string) ([]byte, error) {
	contentPath := filepath.Join(rootDir, path)
	if staticMode {		
		contentPath = strings.ReplaceAll(contentPath, "\\", "/")
		return staticFS.ReadFile(contentPath)
	}
	return os.ReadFile(contentPath)
}

// processImageTags traverses HTML nodes and processes src attributes of img tags.
// file:// paths are changed to URLs served by the server,
// and relative path images are converted to Base64 data URIs and embedded in HTML.
func processImageTags(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "img" {
		for i, attr := range n.Attr {
			if attr.Key == "src" {
				src := attr.Val
				// Handle local file paths starting with file:// protocol
				if strings.HasPrefix(src, "file://") {
					n.Attr[i].Val = convertFileSrc(src)
				} else if !strings.HasPrefix(src, "data:") && !strings.HasPrefix(src, "http") {
					// Embed relative path images by encoding them as Base64
					imageData, err := readFile(src)
					if err != nil {
						log.Printf("⚠️  Could not read image file for embedding %s: %v", src, err)
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
		processImageTags(c)
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
