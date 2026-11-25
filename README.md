# quevu-examples

A small application that shows every part of
[quevu](https://github.com/pablo-botella/quevu) at work: four pieces of
different natures behind one panel, each with a log of its own going to a
different kind of destination.

```
quevu-examples                  the application with its own window
quevu-examples -headless        the same application with no window
quevu-examples -serve demo      also reachable by a separate controller
```

The window is built into the executable. Built without cgo, or for a
system the window does not exist for, it runs headless by itself: the
visual part is optional. Ctrl+C, or the stop order of the system, performs the
same orderly stop the Exit entry of the window does.

## Build

This repository has no releases and promises no interface: it is code to
read, run and copy from. Clone it and build it:

```
git clone https://github.com/pablo-botella/quevu-examples
cd quevu-examples
go build .            # the demonstration application
go build ./quevuctl   # quevuctl
```

Its `go.mod` takes the quevu modules from the folders next to it, so
clone those beside this one.

The window is the `quevu-ui` package, C that cgo builds into the
executable: on Windows a C compiler (gcc from MinGW-w64) must be in the
`PATH`. There is no library to ship: each executable runs alone.

## What is in it

| Unit | Nature | What stopping means |
|---|---|---|
| `web` | A server attending a port | Accept nothing new, finish requests in flight |
| `mailer` | A worker with a queue | Finish the message in hand |
| `rates` | A periodic task | Immediate, unless it happens to be awake |
| `payments` | A group of goroutines | No new operation starts; every one in flight runs to its end |
| `licensing` | A unit that fails to start | - |

Each has a log, and each log shows a different arrangement of the same
three kinds of piece:

| Log | Persistent writers | Immediate provider | Historical provider | So its console... |
|---|---|---|---|---|
| `web` | A text file | A reader over that file | None | goes back into the file by itself |
| `mailer` | SQLite, committed in groups | A reader over the live table, limited to the last day and short searches | None | goes back a day by itself |
| `rates` | None | None | None | ends where the screen's memory ends |
| `payments` | A text file synced to disk, and SQLite | None | An aggregator over two SQLite tables | stops after a hundred lines and offers the history |

Every log also has a volatile writer: the screen of the panel when
somebody can be watching, and a three-line writer that prints to the
real console when nobody can (`-headless` with no `-serve`). The log
itself is the same in both cases; only its list of writers changes.

## Things to try

- **Stop `payments`** and read its console: it says the stop was
  requested, finishes the operations in flight, and only then reports
  stopped. That wait is the point of quevu.
- **Scroll a console back.** It pauses, and keeps going back past what the
  log remembers into the stored file, across restarts of the application.
  `rates` has no store: its way back ends at the short memory.
- **Scroll `payments` back** to its first line: the console stops, says
  there is a history and shows a History button. Nothing is read from the
  database until you press it. Search there for something old and the
  same happens: recent output first, then the offer.
- **Press End** or the play button: it jumps to the live output.
- **Search:** Ctrl+F, type `declined`, Enter. F3 and Shift+F3 walk to
  older and newer matches, out of the screen's memory and into the file.
  In `mailer` a step stops after 20,000 lines because its provider says
  so.
- **Press the three dots** next to the search arrows: a row folds out with
  a range of time to search within.
- **Console > Show the time of each line.**
- **Close the window:** with the tray icon on, it hides. Exit is in the
  menu, and stops everything in order before the process ends.
- **Run it with `-serve demo`** and start `quevuctl demo`, built from the
  `quevuctl` folder here: a second window on the same application,
  through a pipe.
- **Run it with `-headless -serve demo`**: no window at all, and the
  controller still works.

## Flags

| Flag | Meaning |
|---|---|
| `-data dir` | Directory for the persistent logs (default `data`) |
| `-headless` | No window |
| `-serve name` | Local endpoint a separate controller can connect to |

## Rules and gotchas

- **The window is built into the executable** (`quevu-ui`, C compiled by
  cgo): building needs a C compiler. Built without cgo the demo says so
  and runs headless.
- **The `web` unit listens on a free local port** chosen at start, shown
  in its console and opened by the *Open the web status page* menu entry.
- **Nothing here is a real mail or payment**: the pieces sleep and print.

## quevuctl: the panel of an application running elsewhere

In the folder `quevuctl` there is a second program: the panel for an
application that runs somewhere else on this machine, typically as a
service with no window of its own.

```
go build ./quevuctl
quevuctl billing
```

`billing` is the name the application listens on (see `quevu-wire`). The
window is the same one an application shows for itself: its switches and
its screens.

It is an example because there is nothing in it: a `quevu-wire` client
shown with `quevu-ui`, in one short `main.go`. It also works as it is,
for any application that serves its panel with `quevu-wire`; a product of
its own would copy it and give it a name and an icon.

The window is the `quevu-ui` package, built into the executable: there is
no library to put next to it.
### How it behaves

- **Closing the window detaches.** It is not an order for the application,
  which keeps running. To stop the application, use its switches.
- **The switches are real.** Stop on a unit asks the application to stop
  it in order, and the switch shows *stopping* until the application says
  it is done.
- **Several controllers can watch the same application**, and each sees
  what the others do.
- **When the application goes away, so does the window**, with a line on
  standard error saying so.
- **Providers work across the connection.** Whatever the application
  joined to the screen of a log (`quevu.Join`) is reachable from here,
  on the same terms: recent output by itself, the history only when you
  press History.

### Flags of quevuctl

| Flag | Meaning |
|---|---|
| `-wait d` | How long to wait for the application to answer (default 5s) |

### Rules and gotchas of quevuctl

- **It needs the window**: a controller without one has
  nothing to do, so unlike an application it does not fall back to
  running headless.
- **Local only.** The endpoint is a named pipe on Windows and a Unix
  domain socket elsewhere, reachable by the user that created it. The
  controller must run as the same user as the application.
- **Starting and stopping the service itself** (the process) is the
  business of the service manager of the system. quevuctl talks to an
  application that is already running.

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

## License

MIT - see [LICENSE](LICENSE).
