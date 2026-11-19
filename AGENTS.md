# quevu-demo - agent notes

Go command `github.com/pablo-botella/quevu-demo`: a demonstration
application that wires every quevu module together. `main.go` is the
wiring and reads top to bottom as a worked example; `units.go` holds the
four fake pieces. It is the place to try a change to any quevu module end
to end.

## What is in it

| Unit | Nature | What stopping means | Its log goes to |
|---|---|---|---|
| `web` | A server attending a port | Accept nothing new, finish requests in flight | A text file |
| `mailer` | A worker with a queue | Finish the message in hand | SQLite, committed in groups |
| `rates` | A periodic task | Immediate, unless it happens to be awake | The console only |
| `payments` | A group of goroutines | No new operation starts; every one in flight runs to its end | A text file synced to disk, and SQLite, at once |
| `licensing` | A unit that fails to start | - | - |

The four logs cover the range: output that is only looked at, output on
the fast lane, output on the safe lane, and output going to two devices at
the same time, each at its own speed.

## Things to try

- **Stop `payments`** and read its console: it says the stop was
  requested, finishes the operations in flight, and only then reports
  stopped. That wait is the point of quevu.
- **Scroll a console back.** It pauses, and keeps going back past what the
  log remembers into the stored file, across restarts of the application.
  `rates` has no store: its way back ends at the short memory.
- **Press End** or Resume: it jumps to the live output.
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
