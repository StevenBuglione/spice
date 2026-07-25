package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	hello "github.com/StevenBuglione/spice/internal/spicegen/hello"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Printf("Spice example failed: %v", err)
		os.Exit(1) // Entrypoint exception: return a non-zero status when the server cannot run.
	}
}

func run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("hello-world", flag.ContinueOnError)
	flags.SetOutput(stdout)
	check := flags.Bool("check", false, "construct the generated application, print its route, and exit")
	if err := flags.Parse(arguments); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	application, err := hello.NewApplication(context.Background())
	if err != nil {
		return fmt.Errorf("construct application: %w", err)
	}
	if *check {
		if err := application.Stop(context.Background()); err != nil {
			return fmt.Errorf("release application: %w", err)
		}
		if _, err := fmt.Fprintln(stdout, "Spice example ready: GET /users/{id}"); err != nil {
			return fmt.Errorf("write readiness message: %w", err)
		}
		return nil
	}

	runContext, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()
	return application.Run(runContext, func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), shutdownTimeout)
	})
}
