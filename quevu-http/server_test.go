package qhttp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func get(t *testing.T, c *http.Client, url string) (int, string, error) {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), nil
}

func TestRequestsInFlightFinish(t *testing.T) {
	entered := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		time.Sleep(150 * time.Millisecond)
		io.WriteString(w, "whole answer")
	})
	s := &Server{Addr: "127.0.0.1:0", Handler: mux}
	if s.Address() != "" || s.Done() != nil {
		t.Fatal("running before being started")
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatal("stopping what never started:", err)
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	addr := s.Address()
	if addr == "" || strings.HasSuffix(addr, ":0") {
		t.Fatalf("address %q", addr)
	}
	if err := s.Start(context.Background()); !errors.Is(err, ErrRunning) {
		t.Fatalf("a second start: %v", err)
	}
	// Another server on the same port: an error of its start, nothing more.
	if err := (&Server{Addr: addr, Handler: mux}).Start(context.Background()); err == nil {
		t.Fatal("two servers on one port")
	}

	got := make(chan string, 1)
	go func() {
		_, body, err := get(t, http.DefaultClient, "http://"+addr+"/slow")
		if err != nil {
			body = "error: " + err.Error()
		}
		got <- body
	}()
	<-entered
	done := s.Done()
	if err := s.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if body := <-got; body != "whole answer" {
		t.Fatalf("the request in flight got %q", body)
	}
	if err, ok := <-done; err != nil || !ok {
		t.Fatalf("done said %v", err)
	}
	if s.Address() != "" {
		t.Fatal("an address after stopping")
	}
	if _, _, err := get(t, &http.Client{Timeout: time.Second}, "http://"+addr+"/slow"); err == nil {
		t.Fatal("still listening after the stop")
	}
}

func TestAnImpatientStopCutsNothing(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	var answered atomic.Bool
	s := &Server{Addr: "127.0.0.1:0", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		io.WriteString(w, "late but whole")
		answered.Store(true)
	})}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	addr := s.Address()
	got := make(chan string, 1)
	go func() {
		_, body, _ := get(t, http.DefaultClient, "http://"+addr+"/")
		got <- body
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err := s.Stop(ctx)
	cancel()
	if err == nil || answered.Load() {
		t.Fatalf("an impatient stop: %v", err)
	}
	// The request goes on and is answered whole.
	close(release)
	if body := <-got; body != "late but whole" {
		t.Fatalf("the request got %q", body)
	}
	// And a patient stop now finishes the job.
	if err := s.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.Address() != "" {
		t.Fatal("still running")
	}
}

func TestItStartsAgainAndCanBeConfigured(t *testing.T) {
	var configured atomic.Int32
	s := &Server{Addr: "127.0.0.1:0", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "hi") }),
		Configure: func(srv *http.Server) {
			configured.Add(1)
			srv.ReadHeaderTimeout = time.Second
			if srv.Handler == nil {
				panic("configured before being set up")
			}
		}}
	for i := 1; i <= 3; i++ {
		if err := s.Start(context.Background()); err != nil {
			t.Fatal(i, err)
		}
		if code, body, err := get(t, http.DefaultClient, "http://"+s.Address()+"/"); err != nil || code != 200 || body != "hi" {
			t.Fatal(i, code, body, err)
		}
		if err := s.Stop(context.Background()); err != nil {
			t.Fatal(i, err)
		}
	}
	if configured.Load() != 3 {
		t.Fatalf("configured %d times", configured.Load())
	}
}

func TestAServerThatEndsByItselfIsNoticed(t *testing.T) {
	s := &Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := s.Done()
	s.mu.Lock()
	s.ln.Close() // the listener dies under it
	s.mu.Unlock()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("ended by itself with no error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("not noticed")
	}
	if s.Address() != "" {
		t.Fatal("still counted as running")
	}
	// It can be started again.
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.Stop(context.Background())
}

func TestHTTPS(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	pool := x509.NewCertPool()
	leaf, _ := x509.ParseCertificate(der)
	pool.AddCert(leaf)

	s := &Server{Addr: "127.0.0.1:0", TLS: &tls.Config{Certificates: []tls.Certificate{cert}},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "secret") })}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer s.Stop(context.Background())
	c := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
	if code, body, err := get(t, c, "https://"+s.Address()+"/"); err != nil || code != 200 || body != "secret" {
		t.Fatal(code, body, err)
	}
}

func TestObserve(t *testing.T) {
	var mu sync.Mutex
	var lines []string
	var number atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/teapot", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.WriteHeader(http.StatusOK) // too late: the first one counts
		io.WriteString(w, "12345")
		io.WriteString(w, "678")
	})
	mux.HandleFunc("/quiet", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "a")
		if f, ok := w.(http.Flusher); !ok {
			t.Error("the writer lost its Flush")
		} else {
			f.Flush()
		}
		if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(time.Minute)); err != nil {
			t.Error("the controller cannot reach the real writer:", err)
		}
		time.Sleep(30 * time.Millisecond)
		io.WriteString(w, "b")
	})
	h := Observe(mux,
		func(r *http.Request) int64 {
			n := number.Add(1)
			mu.Lock()
			lines = append(lines, fmt.Sprintf("%d in %s", n, r.URL.Path))
			mu.Unlock()
			return n
		},
		func(r *http.Request, n int64, a Answer) {
			// only the handler that sleeps is known to be slow: the others take what the machine takes
			slow := r.URL.Path == "/stream" && a.Took >= 30*time.Millisecond
			if a.Took < 0 { // zero is a real answer: a clock may not see a handler that does nothing
				t.Error("a negative time")
			}
			mu.Lock()
			lines = append(lines, fmt.Sprintf("%d out %s %d %d slow=%v", n, r.URL.Path, a.Status, a.Bytes, slow))
			mu.Unlock()
		})
	s := &Server{Addr: "127.0.0.1:0", Handler: h}
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer s.Stop(context.Background())
	for _, p := range []string{"/teapot", "/quiet", "/stream", "/missing"} {
		if _, _, err := get(t, http.DefaultClient, "http://"+s.Address()+p); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{
		"1 in /teapot", "1 out /teapot 418 8 slow=false",
		"2 in /quiet", "2 out /quiet 200 0 slow=false",
		"3 in /stream", "3 out /stream 200 2 slow=true",
		"4 in /missing", "4 out /missing 404 19 slow=false",
	}
	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(lines) != fmt.Sprint(want) {
		t.Fatalf("\n got  %q\n want %q", lines, want)
	}

	// Either function may be missing, and a handler that panics is still
	// told about before the panic goes on its way.
	var after atomic.Int32
	p := Observe[struct{}](http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }), nil,
		func(*http.Request, struct{}, Answer) { after.Add(1) })
	func() {
		defer func() { recover() }()
		p.ServeHTTP(nil, nil)
	}()
	if after.Load() != 1 {
		t.Fatal("a handler that panics was not told about")
	}
	Observe[int](http.NotFoundHandler(), nil, nil).ServeHTTP(discard{}, &http.Request{})
}

type discard struct{}

func (discard) Header() http.Header         { return http.Header{} }
func (discard) Write(p []byte) (int, error) { return len(p), nil }
func (discard) WriteHeader(int)             {}
