// quevu-demo is a small application that shows every part of quevu at
// work: four pieces of different natures behind one panel, each with a
// log of its own going to a different kind of destination.
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

func run(dataDir string, headless bool, serve, libPath string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	panel := quevu.New()

	// web: a text file, readable with any tool.
	webPath := filepath.Join(dataDir, "web.log")
	webDev, err := qtxt.Open(webPath, qtxt.Options{})
	if err != nil {
		return err
	}
	webLog, err := qlog.New("web", qlog.WithDevice(webDev))
	if err != nil {
		return err
	}
	panel.AddSource(quevu.WithHistory(webLog, qtxt.NewReader(webPath)))

	// mailer: SQLite, committing in groups. Fast lane: what is in the
	// buffer is lost if the process dies.
	dbPath := filepath.Join(dataDir, "demo.db")
	mailDev, err := qsqlite.Open(dbPath, qsqlite.Options{Table: "mailer", Buffer: 50, FlushEvery: time.Second})
	if err != nil {
		return err
	}
	mailLog, err := qlog.New("mailer", qlog.WithDevice(mailDev))
	if err != nil {
		return err
	}
	mailHistory, err := qsqlite.NewReader(dbPath, "mailer")
	if err != nil {
		return err
	}
	panel.AddSource(quevu.WithHistory(mailLog, mailHistory))

	// rates: the console only. Nothing is kept beyond the short memory.
	ratesLog, err := qlog.New("rates")
	if err != nil {
		return err
	}
	panel.AddSource(ratesLog)

	// payments: both at once, and the text file on the safe lane, synced
	// to disk batch by batch.
	payPath := filepath.Join(dataDir, "payments.log")
	payTxt, err := qtxt.Open(payPath, qtxt.Options{Sync: true})
	if err != nil {
		return err
	}
	payDB, err := qsqlite.Open(dbPath, qsqlite.Options{Table: "payments"})
	if err != nil {
		return err
	}
	payLog, err := qlog.New("payments", qlog.WithDevice(payTxt), qlog.WithDevice(payDB))
	if err != nil {
		return err
	}
	panel.AddSource(quevu.WithHistory(payLog, qtxt.NewReader(payPath)))

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

	var lib *qui.Library
	if !headless {
		lib, err = qui.Load(libPath)
		if err != nil {
			if !qui.IsNoLibrary(err) {
				return err
			}
			fmt.Println("no visual part on this machine, running headless:", err)
		}
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
