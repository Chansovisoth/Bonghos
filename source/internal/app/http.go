package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/auth"
	"github.com/Chansovisoth/Bonghos/internal/authorization"
)

const (
	sessionCookie = "bonghos_session"
	csrfCookie    = "bonghos_csrf"
	csrfHeader    = "X-Bonghos-CSRF"
)

type ctxKey int

const ctxUser ctxKey = 1

// currentUser returns the authenticated user attached by requireAuth.
func currentUser(r *http.Request) *auth.User {
	u, _ := r.Context().Value(ctxUser).(*auth.User)
	return u
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func readJSON(r *http.Request, v any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = 1 << 20
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON value")
		}
		return err
	}
	return nil
}

// secureHeaders sets standard security headers and a CSP suitable for the
// embedded single-page frontend.
func (a *App) secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Robots-Tag", "noindex, nofollow, noarchive, nosnippet, noimageindex")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), publickey-credentials-create=(self), publickey-credentials-get=(self)")
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data: https://minotar.net https://cdn.discordapp.com; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self'; connect-src 'self' ws: wss:; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// csrfProtect enforces double-submit cookie CSRF on mutating requests.
func (a *App) csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		c, err := r.Cookie(csrfCookie)
		hdr := r.Header.Get(csrfHeader)
		if err != nil || hdr == "" ||
			subtle.ConstantTimeCompare([]byte(c.Value), []byte(hdr)) != 1 {
			writeErr(w, http.StatusForbidden, errors.New("CSRF verification failed"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) issueCSRF(w http.ResponseWriter, r *http.Request) string {
	var tok string
	existing := false
	if c, err := r.Cookie(csrfCookie); err == nil && len(c.Value) == 64 {
		tok = c.Value
		existing = true
	} else {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		tok = hex.EncodeToString(b)
	}
	secure := requestUsesHTTPS(r)
	// Same-origin GET requests made through a TLS-terminating tunnel may omit
	// Origin. If a valid cookie already exists, do not reissue it without the
	// Secure attribute and accidentally downgrade the cookie created at login.
	// A later HTTPS POST still reissues it as Secure, which also upgrades a token
	// first obtained from the unauthenticated login page.
	if existing && !secure {
		return tok
	}
	http.SetCookie(w, &http.Cookie{
		Name: csrfCookie, Value: tok, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: secure,
	})
	return tok
}

// requireAuth validates the session cookie and attaches the user.
func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := a.sessionUser(r)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, errors.New("authentication required"))
			return
		}
		ctx := context.WithValue(r.Context(), ctxUser, u)
		next(w, r.WithContext(ctx))
	}
}

func (a *App) sessionUser(r *http.Request) (*auth.User, error) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil, errors.New("no session")
	}
	return a.Auth.ValidateSession(c.Value)
}

// requirePerm layers a granular permission check over requireAuth.
func (a *App) requirePerm(p authorization.Permission, next http.HandlerFunc) http.HandlerFunc {
	return a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		u := currentUser(r)
		if u == nil || !authorization.Has(u.Role, p) {
			writeErr(w, http.StatusForbidden, errors.New("permission denied"))
			return
		}
		next(w, r)
	})
}

func (a *App) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
		Secure: requestUsesHTTPS(r), Expires: expires,
	})
}

func (a *App) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/",
		HttpOnly: true, Secure: requestUsesHTTPS(r), MaxAge: -1, SameSite: http.SameSiteStrictMode,
	})
}

// requestUsesHTTPS recognizes direct TLS and a same-origin HTTPS request made
// through a local reverse proxy or tunnel. It deliberately does not trust
// forwarding headers, which an exposed client could forge when Bonghos is
// bound directly.
func requestUsesHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	origin := r.Header.Get("Origin")
	u, err := url.Parse(origin)
	return err == nil && u.Scheme == "https" && strings.EqualFold(u.Host, r.Host)
}

// spaHandler serves the embedded frontend, falling back to index.html for
// client-side routes.
func (a *App) spaHandler() http.Handler {
	if a.WebFS == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html><title>Bonghos</title><p>Frontend not embedded in this build. API is available under /api.</p>"))
		})
	}
	fileServer := http.FileServer(http.FS(a.WebFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(a.WebFS, p); err != nil {
			// SPA fallback
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func remoteIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}
