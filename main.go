package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
)

const showLog = true

var staticMode = false

//go:embed embed
var embeddedFS embed.FS

//go:embed static/**
var staticFS embed.FS

var contentFS fs.FS
var handlerFS http.FileSystem
var rootDir string
var staticServer http.Handler

// createListener creates a listener on an available port
func createListener() (net.Listener, error) {
	return net.Listen("tcp", "localhost:0")
}

func main() {
	log.Printf("Development mode: %v", IsDev)

	// Check if static/index.html exists and set staticMode
	if _, err := staticFS.Open("static/index.html"); err == nil {
		staticMode = true
		log.Println("💡 Found static/index.html. Serving from embedded static assets.")
	}

	var htmlFilePath string
	if !staticMode {
		if len(os.Args) < 2 {
			fmt.Println("Usage: gohta <path-to-html-file-or-directory>")
			return
		}
		htmlFilePath = os.Args[1]
	}

	if staticMode {
		rootDir = "static"
		subFS, err := fs.Sub(staticFS, rootDir)
		if err != nil {
			log.Fatalf("❌ Failed to create sub-filesystem for static assets: %v", err)
		}
		contentFS = subFS
		handlerFS = http.FS(subFS)
	} else {
		info, err := os.Stat(htmlFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Fatalf("❌ Error: Input path does not exist: %s", htmlFilePath)
			}
			log.Fatalf("❌ Error checking input path: %v", err)
		}

		if info.IsDir() {
			htmlFilePath = filepath.Join(htmlFilePath, "index.html")
			if _, err := os.Stat(htmlFilePath); err != nil {
				log.Fatalf("❌ Error: index.html not found in directory: %v", err)
			}
		}

		absPath, err := filepath.Abs(htmlFilePath)
		if err != nil {
			log.Fatalf("❌ Error getting absolute path for file: %v", err)
		}
		rootDir = filepath.Dir(absPath)
		contentFS = os.DirFS(rootDir)
		handlerFS = http.Dir(rootDir)
	}
	staticServer = http.FileServer(handlerFS)

	var windowSizeArg string
	var content []byte
	var err error
	if staticMode {
		content, err = staticFS.ReadFile("static/index.html")
	} else {
		content, err = os.ReadFile(htmlFilePath)
	}
	if err != nil {
		log.Fatalf("❌ Error reading file for window size check: %v", err)
	}
	width, height := findGohtaOptions(string(content))
	if width != "" && height != "" {
		windowSizeArg = fmt.Sprintf("--window-size=%s,%s", width, height)
		fmt.Printf("💡 Found gohta:application tag. Setting window size to %sx%s\n", width, height)
	}

	// Temporary Chrome profile directory. Cleaned up when app exits.
	tempDir := filepath.Join(os.TempDir(), "gohta-chrome-profile")
	defer os.RemoveAll(tempDir)

	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		staticServer.ServeHTTP(w, r)
		//http.ServeFile(w, r, filepath.Join(rootDir, "favicon.ico"))
	})
	mux.HandleFunc("/", htmlHandler())
	mux.HandleFunc("/api/", apiHandler)
	mux.HandleFunc("/file/", fileHandler)

	// Serve embedded files
	embedDir, err := fs.Sub(embeddedFS, "embed")
	if err != nil {
		log.Fatalf("❌ Failed to get embed subdirectory: %v", err)
	}
	mux.Handle("/embed/", http.StripPrefix("/embed/", http.FileServer(http.FS(embedDir))))

	// Initialize development mode if enabled
	if IsDev {
		initDevMode(mux, rootDir)
	}

	// Create listener on available port
	listener, err := createListener()
	if err != nil {
		log.Fatalf("❌ Error creating listener: %v", err)
	}
	defer listener.Close()

	// Extract port number
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Configure server to only accept localhost connections
	server := &http.Server{
		Handler:      ternary(showLog, loggingMiddleware(mux), http.Handler(mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in goroutine
	go func() {
		fmt.Printf("🚀 Server running at http://localhost:%d...\n", port)
		fmt.Printf("   - Home: http://localhost:%d/\n", port)
		fmt.Printf("   - Health: http://localhost:%d/health\n", port)
		fmt.Printf("   - API: http://localhost:%d/api\n", port)

		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for the server to start by probing the port
	for i := 0; i < 100; i++ { // Limit retries to prevent infinite loop
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err == nil {
			conn.Close()
			break // Server is ready
		}
		time.Sleep(20 * time.Millisecond)
		if i == 99 {
			log.Fatalf("❌ Failed to start server: could not connect after several retries")
		}
	}

	// Open in Chrome app mode
	url := fmt.Sprintf("http://localhost:%d/app/", port)
	cmd, err := openChromeAppMode(url, tempDir, windowSizeArg)
	if err != nil {
		log.Printf("⚠️ Failed to run Chrome app mode: %v", err)
		log.Printf("Please open %s directly in your browser.", url)
		// Wait to prevent server from exiting when Chrome fails to start
		select {}
	} else {
		fmt.Println("🌐 Browser opened in Chrome app mode!")
		fmt.Println("💡 Server will automatically close when browser window is closed.")
	}

	// Set up a channel to listen for OS signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine to handle cleanup on shutdown
	go func() {
		<-c
		log.Println("🔌 Interrupt signal received. Shutting down...")
		if cmd != nil && cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("❌ Failed to kill Chrome process: %v", err)
			}
		}
		os.Exit(0)
	}()

	// Wait for browser process to exit
	if err := cmd.Wait(); err != nil {
		log.Printf("Error waiting for browser process: %v", err)
	}

	fmt.Println("👋 Browser closed. Shutting down server...")

	// Graceful server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}

	log.Println("Server shutdown successfully.")
}

// findGohtaOptions parses HTML content to find width and height from gohta tag.
func findGohtaOptions(htmlContent string) (width, height string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Warning: Could not parse HTML to find gohta:application options: %v", err)
		return "", ""
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		// Extract width and height values when gohta:application tag is found
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "gohta:application" {
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "width":
					width = a.Val
				case "height":
					height = a.Val
				}
			}
			return // Exit traversal after finding first tag
		}

		// Recursively search child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
			// No need to search further if both width and height are found
			if width != "" && height != "" {
				return
			}
		}
	}
	f(doc)
	return width, height
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("📥 %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("✅ %s %s completed in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Open URL in Chrome app mode
func openChromeAppMode(url string, tempDir string, windowSizeArg string) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	args := []string{
		"--app=" + url,
		// "--window-size=800,600", // Commented out as it's handled dynamically
		"--user-data-dir=" + tempDir, // Use isolated profile
		"--no-first-run",
		"--no-default-browser-check",
	}
	if windowSizeArg != "" {
		args = append(args, windowSizeArg)
	}

	switch runtime.GOOS {
	case "windows":
		// Search for Chrome executable path on Windows
		chromePaths := []string{
			os.ExpandEnv(`$ProgramFiles\Google\Chrome\Application\chrome.exe`),
			os.ExpandEnv(`$ProgramFiles (x86)\Google\Chrome\Application\chrome.exe`),
			os.ExpandEnv(`$LocalAppData\Google\Chrome\Application\chrome.exe`),
		}
		chromePath := "chrome" // Default to PATH search
		for _, path := range chromePaths {
			if _, err := os.Stat(path); err == nil {
				chromePath = path
				break
			}
		}
		cmd = exec.Command(chromePath, args...)

	case "darwin":
		// Specify direct executable path on macOS
		chromePath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		if _, err := os.Stat(chromePath); os.IsNotExist(err) {
			// Fallback to 'open' command if direct path not found
			openArgs := []string{"-a", "Google Chrome", url}
			openArgs = append(openArgs, "--args")
			openArgs = append(openArgs, args[1:]...)
			return exec.Command("open", openArgs...), nil
		}
		cmd = exec.Command(chromePath, args...)

	case "linux":
		// Linux
		cmd = exec.Command("google-chrome", args...)

	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
