---
mkskill:
  pos: 10
  in: readme
---

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

With no `quevu-ui` library around it runs headless by itself: the visual
part is optional. Ctrl+C, or the stop order of the system, performs the
same orderly stop the Exit entry of the window does.
