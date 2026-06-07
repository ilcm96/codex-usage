package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ilcm96/codex-usage/internal/server/httpapi"
)

const sessionCookieName = "codex_usage_session"

type SessionManager struct {
	secret []byte
	ttl    time.Duration
	secure bool
}

func NewSessionManager(secret []byte, ttl time.Duration, secure bool) SessionManager {
	return SessionManager{
		secret: secret,
		ttl:    ttl,
		secure: secure,
	}
}

func (m SessionManager) Set(w http.ResponseWriter) error {
	nonce, err := randomToken(24)
	if err != nil {
		return fmt.Errorf("create session nonce: %w", err)
	}

	expiresAt := time.Now().Add(m.ttl).Unix()
	value := fmt.Sprintf("admin:%d:%s", expiresAt, nonce)
	signed := value + "." + m.sign(value)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    signed,
		Path:     "/",
		Expires:  time.Unix(expiresAt, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
	})
	return nil
}

func (m SessionManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
	})
}

func (m SessionManager) Valid(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return false
	}

	value, sig, ok := strings.Cut(c.Value, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(m.sign(value))) {
		return false
	}

	parts := strings.Split(value, ":")
	if len(parts) != 3 || parts[0] != "admin" {
		return false
	}
	expiresAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() < expiresAt
}

func (m SessionManager) sign(value string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func RequireSession(sm SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sm.Valid(r) {
			httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireDeviceToken(tokens map[string]struct{}, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.Header.Get("X-Device-Token"))
		if token == "" {
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			}
		}
		if _, ok := tokens[token]; !ok {
			httpapi.WriteError(w, http.StatusUnauthorized, "invalid device token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
