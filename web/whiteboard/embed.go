package whiteboard

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed css js *.html
var whiteboardFS embed.FS

// Handler returns an http.Handler that serves the embedded whiteboard files
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		// Strip /whiteboard prefix to get the file path
		cleanPath := strings.TrimPrefix(r.URL.Path, "/whiteboard")
		cleanPath = strings.TrimPrefix(cleanPath, "/")
		cleanPath = path.Clean(cleanPath)
		
		if cleanPath == "." || cleanPath == "" {
			cleanPath = "index.html"
		}

		// Check if file exists
		if _, err := fs.Stat(whiteboardFS, cleanPath); err != nil {
			// If not found, serve index.html for SPA-like behavior
			cleanPath = "index.html"
		}

		// Serve the file
		data, err := whiteboardFS.ReadFile(cleanPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Set content type based on file extension
		contentType := "text/html"
		if strings.HasSuffix(cleanPath, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(cleanPath, ".js") {
			contentType = "application/javascript"
		}

		w.Header().Set("Content-Type", contentType)
		w.Write(data)
	})
}
