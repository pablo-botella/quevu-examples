package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pablo-botella/quevu"
)

// The four pieces below are of four different natures on purpose. What
// they share is only the quevu.Unit they are wrapped in: each one decides
// for itself what "stop" means.

// web is a server attending a port. Stopping it means: accept nothing
// new, let the requests in flight finish.
type web struct {
	out  io.Writer
	addr atomic.Value // string
	srv  *http.Server
	stop context.CancelFunc
	wg   sync.WaitGroup
}

func (w *web) Addr() string {
	a, _ := w.addr.Load().(string)
	return a
}

func (w *web) Start(context.Context) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	w.addr.Store(ln.Addr().String())
	mux := http.NewServeMux()
	var served atomic.Int64
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		began := time.Now()
		time.Sleep(time.Duration(20+rand.IntN(120)) * time.Millisecond)
		n := served.Add(1)
		fmt.Fprintf(rw, "quevu demo\nrequests served: %d\n", n)
		fmt.Fprintf(w.out, "%s %s 200 %dms\n", r.Method, r.URL.Path, time.Since(began).Milliseconds())
	})
	w.srv = &http.Server{Handler: mux}
	fmt.Fprintf(w.out, "listening on http://%s\n", ln.Addr())
	go w.srv.Serve(ln)

	// Some traffic of its own, so there is something to look at.
	ctx, cancel := context.WithCancel(context.Background())
	w.stop = cancel
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		paths := []string{"/", "/status", "/orders/17", "/static/app.js", "/health"}
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(150+rand.IntN(600)) * time.Millisecond):
			}
			resp, err := http.Get("http://" + w.Addr() + paths[rand.IntN(len(paths))])
			if err == nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	}()
	return nil
}

func (w *web) Stop(ctx context.Context) error {
	w.stop()
	w.wg.Wait()
	fmt.Fprintln(w.out, "no longer accepting connections; finishing requests in flight")
	err := w.srv.Shutdown(ctx)
	fmt.Fprintln(w.out, "stopped")
	return err
}

// mailer is a worker with a queue and no port. Stopping it in the middle
// of a message is the ugly case: it finishes the one in hand first.
func mailer(out io.Writer) quevu.Unit {
	return quevu.Go(func(ctx context.Context) error {
		fmt.Fprintln(out, "mailer ready")
		for n := 1; ; n++ {
			select {
			case <-ctx.Done():
				fmt.Fprintln(out, "queue closed, nothing in hand: stopped")
				return nil
			case <-time.After(time.Duration(300+rand.IntN(900)) * time.Millisecond):
			}
			to := fmt.Sprintf("user%03d@example.com", rand.IntN(1000))
			fmt.Fprintf(out, "message %d to %s: connecting\n", n, to)
			// From here on the message is in hand and is not abandoned,
			// whatever the context says.
			time.Sleep(time.Duration(400+rand.IntN(800)) * time.Millisecond)
			fmt.Fprintf(out, "message %d to %s: "+green("delivered")+"\n", n, to)
		}
	})
}

// rates is a periodic task: asleep most of the time, awake for an
// instant. Stopping it is immediate unless it happens to be awake.
func rates(out io.Writer, every time.Duration) quevu.Unit {
	return quevu.Go(func(ctx context.Context) error {
		rate := 1.3600
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			rate += (rand.Float64() - 0.5) * 0.004
			fmt.Fprintf(out, "USD/CAD %.4f\n", rate)
			select {
			case <-ctx.Done():
				fmt.Fprintln(out, "stopped")
				return nil
			case <-t.C:
			}
		}
	})
}

// payments is a group of goroutines doing work that must not be cut: a
// pre-authorization followed by its capture. Stopping it means no new
// operation starts and every one in flight runs to its end.
func payments(out io.Writer, workers int) quevu.Unit {
	return quevu.Go(func(ctx context.Context) error {
		var wg sync.WaitGroup
		var seq atomic.Int64
		for w := 1; w <= workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Duration(500+rand.IntN(2500)) * time.Millisecond):
					}
					op := seq.Add(1)
					amount := float64(500+rand.IntN(25000)) / 100
					fmt.Fprintf(out, "worker %d: op %d pre-authorization of %.2f requested\n", w, op, amount)
					time.Sleep(time.Duration(600+rand.IntN(900)) * time.Millisecond)
					if rand.IntN(12) == 0 {
						fmt.Fprintf(out, "worker %d: op %d "+red("declined")+" by the issuer\n", w, op)
						continue
					}
					fmt.Fprintf(out, "worker %d: op %d authorized, capturing\n", w, op)
					time.Sleep(time.Duration(600+rand.IntN(900)) * time.Millisecond)
					fmt.Fprintf(out, "worker %d: op %d "+green("captured")+"\n", w, op)
				}
			}()
		}
		<-ctx.Done()
		fmt.Fprintln(out, yellow("stop requested: no new operations, waiting for the ones in flight"))
		wg.Wait()
		fmt.Fprintln(out, "all operations finished: stopped")
		return nil
	})
}

// The pieces color their output the way programs written for a terminal
// do, with ANSI escape sequences. The console draws the colors; the stored
// logs keep the sequences as written.
func red(s string) string    { return "\x1b[31m" + s + "\x1b[0m" }
func green(s string) string  { return "\x1b[32m" + s + "\x1b[0m" }
func yellow(s string) string { return "\x1b[33m" + s + "\x1b[0m" }

// flaky fails to start: a unit in trouble has to be visible too.
func flaky() quevu.Unit {
	return quevu.Func(
		func(context.Context) error { return errors.New("no license file") },
		nil,
	)
}
