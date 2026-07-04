package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

type CSRFToken struct {
	Token  string
	Expiry time.Time
}

type CSRFManager struct {
	mu     sync.RWMutex
	tokens map[string]*CSRFToken
	ttl    time.Duration
}

func NewCSRFManager(ttl time.Duration) *CSRFManager {
	m := &CSRFManager{
		tokens: make(map[string]*CSRFToken),
		ttl:    ttl,
	}
	go m.cleanup()
	return m
}

func (m *CSRFManager) GenerateToken(w http.ResponseWriter) string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	m.mu.Lock()
	m.tokens[token] = &CSRFToken{Token: token, Expiry: time.Now().Add(m.ttl)}
	m.mu.Unlock()

	w.Header().Set("X-CSRF-Token", token)
	return token
}

func (m *CSRFManager) ValidateToken(token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, exists := m.tokens[token]
	if !exists {
		return false
	}
	if time.Now().After(t.Expiry) {
		delete(m.tokens, token)
		return false
	}
	delete(m.tokens, token)
	return true
}

func (m *CSRFManager) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for k, v := range m.tokens {
			if now.After(v.Expiry) {
				delete(m.tokens, k)
			}
		}
		m.mu.Unlock()
	}
}

func CSRFMiddleware(csrfManager *CSRFManager, safeMethods ...string) func(http.Handler) http.Handler {
	safe := map[string]bool{"GET": true, "HEAD": true, "OPTIONS": true}
	for _, m := range safeMethods {
		safe[m] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if safe[r.Method] {
				next.ServeHTTP(w, r)
				return
			}

			csrfCookie, err := r.Cookie("csrf_token")
			if err != nil {
				http.Error(w, "missing CSRF token", http.StatusForbidden)
				return
			}
			csrfHeader := r.Header.Get("X-CSRF-Token")
			if csrfHeader == "" {
				http.Error(w, "missing X-CSRF-Token header", http.StatusForbidden)
				return
			}
			if csrfCookie.Value != csrfHeader {
				http.Error(w, "CSRF token mismatch", http.StatusForbidden)
				return
			}
			if !csrfManager.ValidateToken(csrfCookie.Value) {
				http.Error(w, "invalid or expired CSRF token", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
