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
	libPath := flag.String("ui", "", "path of the quevu-ui library (default: look for it)")
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
	if err := run(flag.Arg(0), *libPath, *wait); err != nil {
		fmt.Fprintln(os.Stderr, "quevuctl:", err)
		os.Exit(1)
	}
}

func run(name, libPath string, wait time.Duration) error {
	// The library first: without a window there is nothing to do here.
	lib, err := qui.Load(libPath)
	if err != nil {
		return err
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
			lib.Quit()
		case <-gone:
		}
	}()
	defer close(gone)

	return lib.Run(name+" - quevu", client,
		// Closing this window is not an order for the application.
		qui.OnClose(func() {}),
	)
}
