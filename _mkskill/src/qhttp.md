---
mkskill:
  pos: 120
  in: "*"
---

## quevu-http: an HTTP server with a switch

In the folder `quevu-http` there is a package, `qhttp`, in one file
(`server.go`). It is here to be copied: it imports only the standard
library and knows nothing of quevu.

An HTTP server that can be **stopped in order**: it stops listening, lets
the requests in flight finish, and only then is it stopped. It can be
started again.

```go
web := &qhttp.Server{Addr: ":8080", Handler: mux}
panel.Add("web", web)   // a quevu panel takes it as it is
```

`http.ListenAndServe` never comes back, so an application built on it can
only be killed, requests and all. `Server` has the two functions of a
[quevu](https://github.com/pablo-botella/quevu) unit, `Start` and `Stop`,
and `Done` for a server that ends by itself; but this package imports only
the standard library and can be used alone.

### The server

| Field | What it is |
|---|---|
| `Addr` | Where it listens, `host:port`. A port of zero takes a free one; `Address()` tells which |
| `Handler` | What serves the requests |
| `TLS` | Serves HTTPS with this configuration (it carries the certificates). Nil: plain HTTP |
| `Configure` | Given every `http.Server` just before it serves: the place for timeouts |

- **`Start`** listens and returns once the port is taken. A port that
  cannot be taken is an error of the start, not the death of the process:
  the application decides what a web server that did not start means.
- **`Stop`** closes the door and waits for the requests in flight to be
  answered. If the time given runs out it returns the error: no new
  requests come in, the ones in flight go on, and the server is not
  stopped. Nothing is cut from here.
- **`Done`** yields the result of the current run: nil after a `Stop`,
  the error when the server ended by itself. A quevu panel uses it to
  show a server that fell.
- It can be started again after a stop, and after ending by itself.

```go
web := &qhttp.Server{
	Addr:    "127.0.0.1:0",
	Handler: mux,
	Configure: func(s *http.Server) { s.ReadHeaderTimeout = 10 * time.Second },
}
if err := web.Start(ctx); err != nil { ... }
fmt.Println("listening on", web.Address())
...
web.Stop(ctx) // requests in flight get their whole answer
```

### What happened to each request: `Observe`

```go
var number atomic.Int64
h := qhttp.Observe(mux,
	func(r *http.Request) int64 { // when it comes in
		n := number.Add(1)
		log.Printf("%d|REQ|%s|%s", n, r.Method, r.URL.Path)
		return n
	},
	func(r *http.Request, n int64, a qhttp.Answer) { // when it is answered
		log.Printf("%d|RES|%s|%s|%d|%d|%d", n, r.Method, r.URL.Path,
			a.Status, a.Took.Milliseconds(), a.Bytes)
	})
```

`before` is called when a request comes in and `after` when it has been
answered, with the status, the size of the body and the time it took.
What `before` returns is handed to `after`, so the two share something:
the number given to the request, for one. Either may be nil.

It keeps nothing of the request by itself. What is noted, and where, is
for whoever passes the functions; a line like the ones above, sent to a
quevu log stored with a format, gives a table with a column for each
field.

### Rules and gotchas of the HTTP server

- **Long-lived connections hold a stop back**: `Stop` waits for requests
  in flight, and a handler that never returns (a stream, a long poll)
  makes it wait for the whole time given. Such handlers should watch the
  context of their request.
- **Hijacked connections are not waited for** (WebSocket): that is how
  `http.Server.Shutdown` works. Close them from the application.
- **`after` runs when the handler returns**, also when it panics; the
  panic then goes on its way.
- **The status noted is the first one written**; 200 when the handler
  wrote none.
- The writer handed to the handler keeps `Flush` and works with
  `http.ResponseController`.
