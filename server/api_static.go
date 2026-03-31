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
	// CSP for the RTK call page:
	// - script-src 'unsafe-eval' 'wasm-unsafe-eval': RTK SDK uses WebAssembly and dynamic evaluation
	// - style-src 'unsafe-inline': RTK UI Kit uses CSS-in-JS (SEC-U4-02)
	// - connect-src *: WebSocket/HTTP connections to RTK servers
	// - worker-src blob: 'self': RTK SDK spawns Web Workers via blob: URLs
	// - media-src *: WebRTC audio/video streams
	// - img-src blob:: camera preview and avatar images
	// - fonts.googleapis.com / fonts.gstatic.com: RTK UI Kit's Google Fonts
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-eval' 'wasm-unsafe-eval'; connect-src *; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' blob: data:; worker-src 'self' blob:; media-src *")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
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
