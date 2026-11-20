# quevu-demo

A small application that shows every part of
[quevu](https://github.com/pablo-botella/quevu-kernel) at work: four pieces of
different natures behind one panel, each with a log of its own going to a
different kind of destination.

```
quevu-demo                  the application with its own window
quevu-demo -headless        the same application with no window
quevu-demo -serve demo      also reachable by a separate controller
```

With no `quevu-ui` library around it runs headless by itself: the visual
part is optional. Ctrl+C, or the stop order of the system, performs the
same orderly stop the Exit entry of the window does.

## Install

```
go install github.com/pablo-botella/quevu-demo@latest
```

The window is the `quevu-ui` library, built from its own repository.
Put it next to the executable, or point `QUEVU_UI_LIB` at it.

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

| Log | Persistent consumers | Immediate provider | Historical provider | So its console... |
|---|---|---|---|---|
| `web` | A text file | A reader over that file | None | goes back into the file by itself |
| `mailer` | SQLite, committed in groups | A reader over the live table, limited to the last day and short searches | None | goes back a day by itself |
| `rates` | None | None | None | ends where the screen's memory ends |
| `payments` | A text file synced to disk, and SQLite | None | An aggregator over two SQLite tables | stops after a hundred lines and offers the history |

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

## License

MIT - see [LICENSE](LICENSE).
