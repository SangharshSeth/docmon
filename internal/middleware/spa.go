package middleware

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// ServeStaticWithIndex serves static files and redirects all unmatched routes to index.html
// This is crucial for SPA routing to work properly and avoid 301 redirect loops
func ServeStaticWithIndex(urlPrefix, staticDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If path starts with API prefix, skip this middleware
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.Next()
			return
		}

		// Try to serve the exact file
		requestedPath := c.Request.URL.Path
		if urlPrefix != "/" {
			// Strip the prefix from the path if it exists
			if len(requestedPath) >= len(urlPrefix) && requestedPath[:len(urlPrefix)] == urlPrefix {
				requestedPath = requestedPath[len(urlPrefix):]
				if requestedPath == "" {
					requestedPath = "/"
				}
			}
		}

		// Clean the path to prevent directory traversal
		requestedPath = filepath.Clean(requestedPath)

		// Construct the file path
		filePath := filepath.Join(staticDir, requestedPath)

		// Check if the file exists
		if _, err := os.Stat(filePath); err == nil {
			// File exists, serve it
			c.File(filePath)
			return
		}

		// If the file doesn't exist, serve index.html for SPA routing
		// This is the key to avoiding redirect loops
		c.File(filepath.Join(staticDir, "index.html"))
	}
}
