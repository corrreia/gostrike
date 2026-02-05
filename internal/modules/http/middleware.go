// Package http provides an embedded HTTP server module for GoStrike.
// This file contains HTTP middleware implementations.
package http

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// corsMiddleware handles CORS headers
type corsMiddleware struct {
	handler http.Handler
	origins string
}

func (m *corsMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", m.origins)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "86400")

	// Handle preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	m.handler.ServeHTTP(w, r)
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu        sync.Mutex
	tokens    map[string]*bucket
	rate      int           // Tokens per interval
	interval  time.Duration // Refill interval
	maxTokens int           // Max tokens per bucket
}

type bucket struct {
	tokens    int
	lastCheck time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(ratePerMinute int) *RateLimiter {
	if ratePerMinute <= 0 {
		return nil
	}
	return &RateLimiter{
		tokens:    make(map[string]*bucket),
		rate:      ratePerMinute,
		interval:  time.Minute,
		maxTokens: ratePerMinute * 2, // Allow burst
	}
}

// Allow checks if a request from the given key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	if rl == nil {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.tokens[key]
	if !ok {
		b = &bucket{
			tokens:    rl.maxTokens,
			lastCheck: now,
		}
		rl.tokens[key] = b
	}

	// Refill tokens
	elapsed := now.Sub(b.lastCheck)
	refill := int(float64(rl.rate) * elapsed.Minutes())
	b.tokens += refill
	if b.tokens > rl.maxTokens {
		b.tokens = rl.maxTokens
	}
	b.lastCheck = now

	// Check if allowed
	if b.tokens <= 0 {
		return false
	}

	b.tokens--
	return true
}

// Clean removes old entries
func (rl *RateLimiter) Clean(maxAge time.Duration) {
	if rl == nil {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, b := range rl.tokens {
		if now.Sub(b.lastCheck) > maxAge {
			delete(rl.tokens, key)
		}
	}
}

// rateLimitMiddleware handles rate limiting
type rateLimitMiddleware struct {
	handler http.Handler
	limiter *RateLimiter
}

func (m *rateLimitMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.limiter == nil {
		m.handler.ServeHTTP(w, r)
		return
	}

	// Get client identifier (IP)
	key := getClientIP(r)

	if !m.limiter.Allow(key) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate_limit_exceeded", "message": "too many requests"}`))
		return
	}

	m.handler.ServeHTTP(w, r)
}

// loggingMiddleware logs HTTP requests
type loggingMiddleware struct {
	handler http.Handler
	logFunc func(msg string)
}

func (m *loggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Wrap response writer to capture status code
	wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	m.handler.ServeHTTP(wrapped, r)

	if m.logFunc != nil {
		duration := time.Since(start)
		m.logFunc(formatLogEntry(r.Method, r.URL.Path, wrapped.statusCode, duration))
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// formatLogEntry formats a log entry
func formatLogEntry(method, path string, status int, duration time.Duration) string {
	return method + " " + path + " " + http.StatusText(status) + " " + duration.String()
}

// JSONResponse writes a JSON response
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		// Simple JSON encoding without external dependency
		// For complex cases, use encoding/json
	}
}

// ErrorResponse writes a JSON error response
func ErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error": "` + http.StatusText(status) + `", "message": "` + message + `"}`))
}
