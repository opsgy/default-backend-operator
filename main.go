package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/opsgy/default-backend-operator/operator"
)

func healthy(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Healthy")
}

func main() {
	var (
		listenAddress          string
		defaultErrorPageFolder string
	)

	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagset.StringVar(&listenAddress, "listen-address", "", "The address the default-backend HTTP server should listen on.")
	flagset.StringVar(&defaultErrorPageFolder, "error-pages", "error_pages", "Folder that contains error pages in the format: 5xx.html, 503.html")

	//nolint: errcheck // Parse() will exit on error.
	flagset.Parse(os.Args[1:])

	operator, err := operator.NewOperator(defaultErrorPageFolder)
	if err != nil {
		log.Fatalf("Failed to setup operator: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthy)
	mux.Handle("/", operator)

	srv := &http.Server{Handler: mux}

	l, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("Failed to listen on address: %v", err)
	}

	errCh := make(chan error)
	go func() {
		log.Printf("Listening on %v", l.Addr())
		errCh <- srv.Serve(l)
	}()

	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		log.Print("Received SIGTERM, exiting gracefully...")
		srv.Close()
	case err := <-errCh:
		if err != http.ErrServerClosed {
			log.Printf("Server stopped with %v", err)
		}
		os.Exit(1)
	}
}
