# gohta
Revived HTA with Go and Chrome

## Development Mode

To enable live reload functionality during development, build with the `dev` tag:

```bash
go build -tags dev
```

### Features in Development Mode

- **File Watching**: Automatically watches for changes in HTML, CSS, JS, JSON, and XML files
- **WebSocket Live Reload**: Establishes WebSocket connection for real-time communication
- **Automatic Browser Refresh**: Browser automatically refreshes when files are modified
- **Debounced Updates**: Prevents multiple rapid reloads when multiple files change

### How It Works

1. **File Watcher**: Uses `fsnotify` library to monitor the directory containing your HTML file
2. **WebSocket Server**: Adds `/ws` endpoint for WebSocket connections
3. **Script Injection**: Automatically injects JavaScript code into HTML pages for WebSocket connection
4. **Change Detection**: When files change, sends "reload" message to all connected browsers
5. **Auto Refresh**: JavaScript receives the message and executes `location.reload()`

### Usage

```bash
# Build in development mode
go build -tags dev

# Run with your HTML file
./gohta your-file.html
```

The browser will automatically refresh whenever you save changes to your HTML, CSS, or JavaScript files.