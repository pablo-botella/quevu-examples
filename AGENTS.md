# quevu-demo - agent notes

Go command `github.com/pablo-botella/quevu-demo`: a demonstration
application that wires every quevu module together. `main.go` is the
wiring and reads top to bottom as a worked example; `units.go` holds the
four fake pieces. It is the place to try a change to any quevu module end
to end.

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

| Log | Persistent consumers | Stored-data provider | So its console... |
|---|---|---|---|
| `web` | A text file | A reader over that file | goes back to the first line ever written |
| `mailer` | SQLite, committed in groups | A reader that gives only the last day and allows short searches | goes back a day |
| `rates` | None | None | ends where the screen's memory ends |
| `payments` | A text file synced to disk, and SQLite | A reader over the file; the database gives nothing back | goes back through the file |

Every log also has a volatile consumer: the screen of the panel when
somebody can be watching, and a three-line consumer that prints to the
real console when nobody can (`-headless` with no `-serve`). The log
itself is the same in both cases; only its list of consumers changes.

## Things to try

- **Stop `payments`** and read its console: it says the stop was
  requested, finishes the operations in flight, and only then reports
  stopped. That wait is the point of quevu.
- **Scroll a console back.** It pauses, and keeps going back past what the
  log remembers into the stored file, across restarts of the application.
  `rates` has no store: its way back ends at the short memory.
- **Press End** or the play button: it jumps to the live output.
- **Search:** Ctrl+F, type `declined`, Enter. F3 and Shift+F3 walk to
  older and newer matches, out of the screen's memory and into the file.
  In `mailer` a step stops after 20,000 lines because its provider says
  so.
- **Console > Show the time of each line.**
- **Close the window:** with the tray icon on, it hides. Exit is in the
  menu, and stops everything in order before the process ends.
- **Run it with `-serve demo`** and start `quevuctl demo` from
  `quevu-ctl`: a second window on the same application, through a pipe.
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
