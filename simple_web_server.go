package main

import (
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

func handleCORS(w http.ResponseWriter) {
	// Set CORS headers to allow requests from any origin
	w.Header().Set("Access-Control-Allow-Origin", "*") // Or use a specific origin instead of "*"
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, ngrok-skip-browser-warning") // Add custom headers here
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight OPTIONS request
		if r.Method == http.MethodOptions {
			handleCORS(w)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Apply CORS headers to all other requests
		handleCORS(w)

		// Serve the file requested
		path := "." + r.URL.Path
		f, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		// Try to get MIME from extension
		mimeType := mime.TypeByExtension(filepath.Ext(path))

		// If unknown, sniff from file
		if mimeType == "" {
			buf := make([]byte, 512)
			n, _ := f.Read(buf)
			mimeType = http.DetectContentType(buf[:n])
			f.Seek(0, 0) // rewind file for actual read
		}

		w.Header().Set("Content-Type", mimeType)
		io.Copy(w, f)
	})

	log.Println("Serving on http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
