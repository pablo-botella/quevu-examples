// quevu-demo is a small application that shows every part of quevu at
// work: four pieces of different natures behind one panel, each with a
// log of its own going to a different mix of consumers.
//
//	quevu-demo                  the application with its own window
//	quevu-demo -headless        the same application with no window
//	quevu-demo -serve demo      also reachable by a separate controller
//
// With no quevu-ui library around it runs headless by itself: the visual
// part is optional. Ctrl+C, or the stop order of the system, performs
// the same orderly stop the Exit entry of the window does.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pablo-botella/quevu"
	qlog "github.com/pablo-botella/quevu-log"
	qpanel "github.com/pablo-botella/quevu-panel"
	qsqlite "github.com/pablo-botella/quevu-sqlite"
	qtxt "github.com/pablo-botella/quevu-txt"
	qui "github.com/pablo-botella/quevu-ui-go"
	qwire "github.com/pablo-botella/quevu-wire"
)

func main() {
	dataDir := flag.String("data", "data", "directory for the persistent logs")
	headless := flag.Bool("headless", false, "run with no window even if the library is there")
	serve := flag.String("serve", "", "name of a local endpoint a separate controller can connect to")
	libPath := flag.String("ui", "", "path of the quevu-ui library (default: look for it)")
	flag.Parse()

	if err := run(*dataDir, *headless, *serve, *libPath); err != nil {
		fmt.Fprintln(os.Stderr, "quevu-demo:", err)
		os.Exit(1)
	}
}

// app is the application being assembled: its panel, and whether its
// logs get a screen.
type app struct {
	panel *quevu.Panel
	// screens says whether anybody can be watching: a window of our own,
	// or a controller that may connect. On a server with neither, the
	// logs simply have one consumer less.
	screens bool
}

// log builds the log of one piece. The division of work shows here:
//
//   - consumers receive the output. Some persist it; the screen, if
//     there is one, keeps a little of it in memory for a console.
//   - roles are the providers a console can turn to beyond what its
//     screen remembers: an immediate one, asked automatically, and a
//     historical one, asked only when the user says so. Having a
//     consumer that stores does not mean having either.
//
// The log itself only knows the list of consumers.
func (a *app) log(name string, memory int, roles quevu.Roles, consumers ...quevu.Consumer) (*qlog.Log, error) {
	var scr *qpanel.Screen
	if a.screens {
		scr = qpanel.New(name, qpanel.WithMemory(memory))
		consumers = append(consumers, scr)
	} else {
		// Nobody can be watching: the output also goes to the real
		// console, through a consumer of three lines.
		consumers = append(consumers, stdout(name))
	}
	opts := make([]qlog.Option, len(consumers))
	for i, c := range consumers {
		opts[i] = qlog.WithConsumer(c)
	}
	l, err := qlog.New(name, opts...)
	if err != nil {
		return nil, err
	}
	// Logs close last, after every unit has stopped.
	a.panel.AddCloser(l)
	if scr != nil {
		scr.ShowStats(l)
		a.panel.AddSource(quevu.Join(scr, roles))
	}
	return l, nil
}

// stdout is a volatile consumer of the simplest kind: it prints what it
// receives and forgets it.
func stdout(name string) quevu.Consumer { return printer(name) }

type printer string

func (p printer) Write(batch []quevu.Chunk) error {
	for _, c := range batch {
		fmt.Printf("%s [%s] %s\n", c.Time.Format("15:04:05"), string(p), quevu.PlainText(nil, c.Data))
	}
	return nil
}

func (printer) Close() error { return nil }

func run(dataDir string, headless bool, serve, libPath string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	var lib *qui.Library
	if !headless {
		var err error
		lib, err = qui.Load(libPath)
		if err != nil {
			if !qui.IsNoLibrary(err) {
				return err
			}
			fmt.Println("no visual part on this machine, running headless:", err)
		}
	}
	a := &app{panel: quevu.New(), screens: lib != nil || serve != ""}
	panel := a.panel

	// web: a text file, readable with any tool, and a reader over it in
	// the immediate role: scrolling back goes into the file by itself.
	webPath := filepath.Join(dataDir, "web.log")
	webFile, err := qtxt.Open(webPath, qtxt.Options{})
	if err != nil {
		return err
	}
	webLog, err := a.log("web", qpanel.DefaultMemory, quevu.Roles{Immediate: qtxt.NewReader(webPath)}, webFile)
	if err != nil {
		return err
	}

	// mailer: SQLite, committing in groups. The fast lane: what is in the
	// buffer is lost if the process dies. Its immediate provider reads
	// the live table and limits what it gives: the last day, and short
	// searches.
	dbPath := filepath.Join(dataDir, "demo.db")
	mailDB, err := qsqlite.Open(dbPath, qsqlite.Options{Table: "mailer", Buffer: 50, FlushEvery: time.Second})
	if err != nil {
		return err
	}
	mailReader, err := qsqlite.NewReader(dbPath, "mailer")
	if err != nil {
		return err
	}
	mailHistory := quevu.Limit(mailReader, quevu.Limits{MaxAge: 24 * time.Hour, SearchBudget: 20_000})
	mailLog, err := a.log("mailer", qpanel.DefaultMemory, quevu.Roles{Immediate: mailHistory}, mailDB)
	if err != nil {
		return err
	}

	// rates: no persistent consumer and no provider. What the screen
	// remembers is all there is.
	ratesLog, err := a.log("rates", qpanel.DefaultMemory, quevu.Roles{})
	if err != nil {
		return err
	}

	// payments: two persistent consumers at once, the text file on the
	// safe lane (synced to disk batch by batch). Nothing reads the file
	// back. The database is read in the historical role, through an
	// aggregator over two tables (an archive that may not exist yet, and
	// the live one): the same kind of reader mailer uses as immediate.
	// Its screen remembers little, so the offer of history shows soon:
	// scroll back past a hundred lines and the console stops and asks.
	payPath := filepath.Join(dataDir, "payments.log")
	payFile, err := qtxt.Open(payPath, qtxt.Options{Sync: true})
	if err != nil {
		return err
	}
	payDB, err := qsqlite.Open(dbPath, qsqlite.Options{Table: "payments"})
	if err != nil {
		return err
	}
	payHistory, err := qsqlite.NewHistory(
		qsqlite.Table{Path: dbPath, Name: "payments_archive"},
		qsqlite.Table{Path: dbPath, Name: "payments"},
	)
	if err != nil {
		return err
	}
	payLog, err := a.log("payments", 100, quevu.Roles{Historical: payHistory}, payFile, payDB)
	if err != nil {
		return err
	}

	site := &web{out: webLog}
	panel.Add("web", site)
	panel.Add("mailer", mailer(mailLog))
	panel.Add("rates", rates(ratesLog, 5*time.Second))
	panel.Add("payments", payments(payLog, 3))
	panel.Add("licensing", flaky())

	if serve != "" {
		ln, err := qwire.Listen(serve)
		if err != nil {
			return err
		}
		defer ln.Close()
		go qwire.Serve(ln, panel)
		fmt.Printf("reachable as %q (%s)\n", serve, qwire.Endpoint(serve))
	}

	if lib == nil {
		fmt.Println("running; Ctrl+C stops in order")
		// One unit of the demo fails to start on purpose, and a real
		// application decides what that means. Here: carry on.
		for _, u := range panel.Units() {
			if err := panel.Start(context.Background(), u.Name); err != nil {
				fmt.Println(err)
			}
		}
		ctx, cancel := signalContext()
		defer cancel()
		<-ctx.Done()
		fmt.Println("stopping in order...")
		return panel.Shutdown()
	}

	panel.StartAll(context.Background())
	err = lib.Run("quevu demo", panel,
		qui.WithTray(),
		qui.WithMenu("Open the web status page", func() {
			if addr := site.Addr(); addr != "" {
				openBrowser("http://" + addr + "/status")
			}
		}),
		qui.WithMenu("Open the data folder", func() { openBrowser(dataDir) }),
	)
	// The window is gone: by now every unit has stopped, and the logs go
	// last.
	return joinErr(err, panel.Shutdown())
}

func joinErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// openBrowser hands a URL or a path to whatever the system opens it with.
func openBrowser(target string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	cmd.Start()
}
