---
mkskill:
  pos: 5
  in: ai*
---

# quevu-examples - agent notes

Go command `github.com/pablo-botella/quevu-examples`: a demonstration
application that wires every quevu module together. `main.go` is the
wiring and reads top to bottom as a worked example; `units.go` holds the
four fake pieces. It is the place to try a change to any quevu module end
to end. The folder `quevuctl` is a second command: a
`quevu-wire` client shown with `quevu/qui`, the panel of an application
running in another process.
