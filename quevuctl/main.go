// quevuctl is the quevu panel for an application that runs somewhere
// else on this machine, typically as a service with no window of its own.
//
//	quevuctl <name>
//
// name is the endpoint the application listens on (see qwire.Listen). The
// window is the same one an application shows for itself; the only
// difference is what closing it means: it detaches and leaves the
// application running. To stop the application, use its switches.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	qwire "github.com/pablo-botella/quevu-wire"
	"github.com/pablo-botella/quevu/qui"
)

func main() {
	wait := flag.Duration("wait", 5*time.Second, "how long to wait for the application to answer")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: quevuctl [flags] <name>")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *wait); err != nil {
		fmt.Fprintln(os.Stderr, "quevuctl:", err)
		os.Exit(1)
	}
}

func run(name string, wait time.Duration) error {
	// The window first: without one there is nothing to do here.
	if !qui.Available() {
		return qui.ErrNoWindow
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	client, err := qwire.Connect(ctx, name)
	if err != nil {
		return fmt.Errorf("cannot reach %q at %s: %w", name, qwire.Endpoint(name), err)
	}
	defer client.Close()

	// A panel with nothing behind it is of no use: when the application
	// goes away, so does this window.
	gone := make(chan struct{})
	go func() {
		select {
		case <-client.Done():
			fmt.Fprintf(os.Stderr, "quevuctl: %q is gone\n", name)
			qui.Quit()
		case <-gone:
		}
	}()
	defer close(gone)

	return qui.Run(name+" - quevu", client,
		// Closing this window is not an order for the application.
		qui.OnClose(func() {}),
	)
}
