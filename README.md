# Gohta
Revived HTA with Go and Chrome

- Build desktop-like apps easily from HTML, similar to HTA.
- When you need more capability, extend it with Go.

## Development Mode

To enable live reload functionality during development, run with the `dev` tag:

```bash
go run -tags dev . your-file.html
```

### Features in Development Mode

- **File Watching**: Automatically watches for changes in HTML, CSS, JS, JSON, and XML files
- **WebSocket Live Reload**: Establishes WebSocket connection for real-time communication
- **Automatic Browser Refresh**: Browser automatically refreshes when files are modified
- **Debounced Updates**: Prevents multiple rapid reloads when multiple files change

### Usage

```bash
# Build in development mode
go build -tags dev

# Run with your HTML file
./gohta your-file.html
```

The browser will automatically refresh whenever you save changes to your HTML, CSS, or JavaScript files.

## Self-Contained App Mode

`gohta` can also run as a self-contained application. By placing your entire website (including `index.html`, CSS, images, etc.) into a `static` directory, `gohta` will automatically embed these files into the executable at build time.

### How It Works

1.  **Create a `static` Directory**: Place all your web assets in a directory named `static` at the root of the project.
2.  **Automatic Detection**: When the application starts, it checks for the existence of a `static/index.html` file within the embedded assets.
3.  **Self-Contained Serving**: If `static/index.html` is found, `gohta` switches to "self-contained mode" and serves all content from the embedded `static` directory. It will no longer require a file path argument.

### Usage

To build a self-contained application with your site embedded:

```bash
# Place your site in the 'static' directory
echo "<h1>Hello from self-contained mode!</h1>" > static/index.html

# Build the application
go build

# Run the self-contained app
./gohta
```

The application will now serve your `index.html` and all other assets from the `static` directory, completely from within the executable.

## Windows Builds: with or without console

On Windows, you can choose whether the app shows a console window.

- With console (default):
```powershell
# Shows console window (useful for logs)
go build
```

- Without console (hide console window):
```powershell
# Build a GUI executable without a console window
go build -ldflags "-H=windowsgui"
```
