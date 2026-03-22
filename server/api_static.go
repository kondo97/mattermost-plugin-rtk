package main

import (
	_ "embed"
	"net/http"
)

//go:embed assets/call.html
var callHTML []byte

//go:embed assets/call.js
var callJS []byte

//go:embed assets/worker.js
var workerJS []byte

// serveCallHTML serves the call page with HTTP security headers.
func (p *Plugin) serveCallHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src *; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	_, _ = w.Write(callHTML)
}

// serveCallJS serves the call JavaScript bundle.
func (p *Plugin) serveCallJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(callJS)
}

// serveWorkerJS serves the Web Worker JavaScript file.
func (p *Plugin) serveWorkerJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(workerJS)
}
