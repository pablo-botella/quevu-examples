// Package qhttp makes an HTTP server a piece that can be stopped in
// order: it stops listening, lets the requests in flight finish, and only
// then is it stopped. It can be started again.
//
// Server has the two functions of a quevu unit (Start and Stop) and the
// one of a unit that may end by itself (Done), so a quevu panel takes it
// as it is; but this package knows nothing of quevu and imports only the
// standard library.
package qhttp

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

// Server is an HTTP server with a switch.
type Server struct {
	// Addr is where it listens, as in net.Listen: "host:port". A port of
	// zero takes a free one; Address tells which.
	Addr string
	// Handler serves the requests.
	Handler http.Handler
	// TLS serves HTTPS with this configuration (it must carry the
	// certificates). Nil serves plain HTTP.
	TLS *tls.Config
	// Configure, when set, is given every http.Server just before it
	// starts serving: the place for timeouts and the rest of its fields.
	// Addr, Handler and TLSConfig are already set.
	Configure func(*http.Server)

	mu   sync.Mutex
	srv  *http.Server
	ln   net.Listener
	done chan error
}

// ErrRunning is what Start answers when the server is already running.
var ErrRunning = errors.New("qhttp: already running")

// Start listens and serves, and returns once the port is taken. A port
// that cannot be taken is an error of the start: nothing dies.
func (s *Server) Start(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != nil {
		return ErrRunning
	}
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: s.Addr, Handler: s.Handler, TLSConfig: s.TLS}
	if s.Configure != nil {
		s.Configure(srv)
	}
	useTLS := srv.TLSConfig != nil
	done := make(chan error, 1)
	s.srv, s.ln, s.done = srv, ln, done
	go func() {
		var err error
		if useTLS {
			err = srv.ServeTLS(ln, "", "")
		} else {
			err = srv.Serve(ln)
		}
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		// A server that ends without being asked is no longer running.
		s.mu.Lock()
		if s.srv == srv {
			s.srv, s.ln = nil, nil
		}
		s.mu.Unlock()
		done <- err
		close(done)
	}()
	return nil
}

// Stop closes the door and waits for the requests in flight to be
// answered. When ctx expires first it returns its error: the server no
// longer accepts requests, the ones in flight go on, and it is not
// stopped. Stopping what is not running is not an error.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	srv := s.srv
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	if s.srv == srv {
		s.srv, s.ln = nil, nil
	}
	s.mu.Unlock()
	return nil
}

// Done yields the result of the current run when it ends: nil after a
// Stop, the error when the server ended by itself. Nil before the first
// Start.
func (s *Server) Done() <-chan error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

// Address returns where the server is listening, "" when it is not. With
// a port of zero in Addr it is the way to learn the port that was taken.
func (s *Server) Address() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

// Answer is how a request was answered.
type Answer struct {
	// Status is the HTTP status. 200 when the handler wrote none.
	Status int
	// Bytes is the size of the body written.
	Bytes int64
	// Took is the time from the request coming in to the handler
	// returning.
	Took time.Duration
}

// Observe wraps a handler to tell what happens to each request: before
// is called when it comes in and after when it has been answered. Either
// may be nil. What before returns is handed to after, so the two can
// share something: the number given to the request, for one.
//
// It keeps nothing of the request by itself: what is noted, and where,
// is for whoever passes the functions.
func Observe[T any](h http.Handler, before func(*http.Request) T, after func(*http.Request, T, Answer)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var note T
		if before != nil {
			note = before(r)
		}
		rec := &recorder{ResponseWriter: w}
		defer func() {
			if after != nil {
				if rec.status == 0 {
					rec.status = http.StatusOK
				}
				after(r, note, Answer{Status: rec.status, Bytes: rec.bytes, Took: time.Since(start)})
			}
		}()
		h.ServeHTTP(rec, r)
	})
}

// recorder notes the status and the size of an answer as it goes out.
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *recorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += int64(n)
	return n, err
}

func (r *recorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the real writer.
func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
