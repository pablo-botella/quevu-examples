---
mkskill:
  pos: 110
  in: "*"
---

## quevuctl: the panel of an application running elsewhere

In the folder `quevuctl` there is a second program: the panel for an
application that runs somewhere else on this machine, typically as a
service with no window of its own.

```
go install github.com/pablo-botella/quevu-examples/quevuctl@latest
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
