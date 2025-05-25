package main

import (
	"fmt"
	"net/http"
)

var (
	Version   = "dev" // default fallback
	Commit    = "none"
	BuildTime = "unknown"
)

func landingPageHandler(w http.ResponseWriter, r *http.Request) {
	info := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head><title>Build Info</title></head>
		<body>
			<h1>Welcome to My Go Server</h1>
			<p><strong>Version:</strong> %s</p>
			<p><strong>Commit:</strong> %s</p>
			<p><strong>Build Time:</strong> %s</p>
		</body>
		</html>`, Version, Commit, BuildTime)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(info))
}
