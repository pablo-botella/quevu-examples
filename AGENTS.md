# quevu-examples - agent notes

Go command `github.com/pablo-botella/quevu-examples`: a demonstration
application that wires every quevu module together. `main.go` is the
wiring and reads top to bottom as a worked example; `units.go` holds the
four fake pieces. It is the place to try a change to any quevu module end
to end. The folder `quevuctl` is a second command: a
`quevu-wire` client shown with `quevu/qui`, the panel of an application
running in another process.

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
| `-headless` | No window, even if the library is there |
| `-serve name` | Local endpoint a separate controller can connect to |
| `-ui path` | Path of the `quevu-ui` library (default: look for it) |

## Rules and gotchas

- **The window needs the `quevu-ui` library** next to the executable or in
  `QUEVU_UI_LIB`. Without it the demo says so and runs headless.
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
shown with `quevu/qui`, in one short `main.go`. It also works as it is,
for any application that serves its panel with `quevu-wire`; a product of
its own would copy it and give it a name and an icon.

The window is the `quevu-ui` library, built from its own repository. Put
it next to the executable, or point `QUEVU_UI_LIB` at it.
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
| `-ui path` | Path of the `quevu-ui` library (default: look for it) |
| `-wait d` | How long to wait for the application to answer (default 5s) |

### Rules and gotchas of quevuctl

- **It needs the `quevu-ui` library**: a controller without a window has
  nothing to do, so unlike an application it does not fall back to
  running headless.
- **Local only.** The endpoint is a named pipe on Windows and a Unix
  domain socket elsewhere, reachable by the user that created it. The
  controller must run as the same user as the application.
- **Starting and stopping the service itself** (the process) is the
  business of the service manager of the system. quevuctl talks to an
  application that is already running.
