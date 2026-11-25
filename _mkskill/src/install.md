---
mkskill:
  pos: 50
  in: readme
---

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
