package editorweb

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Options configures a map editor server.
type Options struct {
	// MapsDir is the directory holding the .tmj files. Required.
	MapsDir string
	// Addr is the listen address. Defaults to 127.0.0.1:7777.
	Addr string
	// Token authenticates API calls. Generated per launch when empty.
	Token string
	// AllowRemote permits non-loopback clients. Off by default: this process
	// writes files inside a git repo in response to HTTP requests.
	AllowRemote bool
	// Strict promotes soft validation issues (missing door targets) to errors.
	Strict bool
	// Anim overrides the animated-tile recording window.
	Anim animOpts
	// Logger receives startup and error output. Defaults to the standard logger.
	Logger *log.Logger
}

// Server is the map editor HTTP server.
type Server struct {
	opts   Options
	store  *MapStore
	schema *cachedSchema
	logger *log.Logger
	mux    http.Handler
}

// New builds a server, deriving and verifying the schema up front.
//
// A probe-derived model that does not match internal/world is fatal here on
// purpose. A selection box silently off by a hitbox height, or a tile drawn from
// a stale frame, is far worse than a server that refuses to start and says why.
func New(opts Options) (*Server, error) {
	if opts.MapsDir == "" {
		return nil, errors.New("editorweb: MapsDir is required")
	}
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:7777"
	}
	if opts.Logger == nil {
		opts.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	if opts.Token == "" {
		tok, err := randomToken()
		if err != nil {
			return nil, err
		}
		opts.Token = tok
	}

	store, err := NewMapStore(opts.MapsDir)
	if err != nil {
		return nil, err
	}
	schema, err := newCachedSchema(opts.Anim)
	if err != nil {
		return nil, fmt.Errorf("editorweb: %w", err)
	}

	s := &Server{opts: opts, store: store, schema: schema, logger: opts.Logger}
	s.mux = s.routes()
	return s, nil
}

// Handler returns the fully wrapped HTTP handler.
func (s *Server) Handler() http.Handler { return s.mux }

// Store exposes the map store for batch tools such as -check.
func (s *Server) Store() *MapStore { return s.store }

// URL is the address to open, including the per-launch token.
func (s *Server) URL() string {
	host := s.opts.Addr
	if strings.HasPrefix(host, "0.0.0.0:") {
		host = "127.0.0.1:" + strings.TrimPrefix(host, "0.0.0.0:")
	}
	return fmt.Sprintf("http://%s/?t=%s", host, s.opts.Token)
}

// ListenAndServe runs until ctx is cancelled, then shuts down gracefully.
func (s *Server) ListenAndServe(ctx context.Context) error {
	srv := &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return err
	}

	// Report the resolved maps directory, because "which tree am I editing" is
	// exactly the question that goes wrong at 1am. The in-game editor resolves
	// assets/maps relative to the CWD and never says so.
	infos, _ := s.store.List()
	s.logger.Printf("map editor: serving %s (%d maps)", s.store.Root, len(infos))
	s.logger.Printf("map editor: open %s", s.URL())

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ---- middleware ----
//
// This server writes files inside the user's repository in response to HTTP
// requests, so "we bound to 127.0.0.1" is not sufficient on its own. The chain
// below is defense in depth against the local-web-server attack classes:
// another user on the machine, DNS rebinding, and cross-origin form posts.

func (s *Server) wrap(h http.Handler) http.Handler {
	return s.recoverPanic(s.loopbackOnly(s.checkHost(s.checkOrigin(s.requireToken(noCache(h))))))
}

func (s *Server) recoverPanic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.logger.Printf("panic serving %s %s: %v", r.Method, r.URL.Path, v)
				writeErr(w, http.StatusInternalServerError, "panic", "internal error")
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// loopbackOnly rejects non-loopback clients independently of the listen address,
// so binding 0.0.0.0 for a LAN test still needs an explicit opt-in.
func (s *Server) loopbackOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.opts.AllowRemote && !isLoopbackAddr(r.RemoteAddr) {
			writeErr(w, http.StatusForbidden, "not_loopback", "this editor only accepts local connections (use -allow-remote to override)")
			return
		}
		h.ServeHTTP(w, r)
	})
}

// checkHost blocks DNS rebinding, where an attacker's domain resolves to
// 127.0.0.1 and the victim's browser is steered into posting here.
func (s *Server) checkHost(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.opts.AllowRemote {
			h.ServeHTTP(w, r)
			return
		}
		host := r.Host
		if hostOnly, _, err := net.SplitHostPort(host); err == nil {
			host = hostOnly
		}
		switch strings.ToLower(host) {
		case "localhost", "127.0.0.1", "::1", "[::1]":
			h.ServeHTTP(w, r)
		default:
			writeErr(w, http.StatusForbidden, "bad_host", "unexpected Host header %q", r.Host)
		}
	})
}

// checkOrigin rejects cross-origin mutations. Combined with the JSON content
// type (which forces a CORS preflight this server never answers), it closes the
// browser-driven attack path. No CORS headers are ever emitted.
func (s *Server) checkOrigin(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && !s.originAllowed(origin) {
			writeErr(w, http.StatusForbidden, "bad_origin", "cross-origin request from %q is not allowed", origin)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// requireToken gates the API on a per-launch secret, so another process or user
// on the same machine cannot drive the editor just by knowing the port.
//
// The index page is exempt: it is what hands the token to the app.
func (s *Server) requireToken(h http.Handler) http.Handler {
	want := []byte(s.opts.Token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			h.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get("X-Editor-Token")
		if got == "" {
			got = r.URL.Query().Get("t")
		}
		if subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			writeErr(w, http.StatusUnauthorized, "bad_token", "missing or invalid editor token")
			return
		}
		h.ServeHTTP(w, r)
	})
}

// noCache keeps the browser from serving a stale map or a stale app build. The
// schema handler opts back in with its own ETag.
func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
