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

The window is the `quevu-ui` library, built from its own repository.
Put it next to the executable, or point `QUEVU_UI_LIB` at it.
