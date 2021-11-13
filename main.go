package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

var Interrupt = errors.New("caught interrupt signal")

func main() {
	ctx := context.Background()
	g, _ := errgroup.WithContext(context.Background())

	// api server
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, "hello world")
	})
	apiServerAddr := ":8080"
	apiServer := http.Server{
		Addr:    apiServerAddr,
		Handler: apiMux,
	}
	shutdownApiServer := func() {
		// wait 30s for server shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		err := apiServer.Shutdown(shutdownCtx)
		if err != nil {
			fmt.Printf("fail to shutdown api server: %v\n", err)
		}
	}

	// interrupt goroutine exit after receive message from this channel
	interruptDone := make(chan struct{})

	// api server goroutine
	g.Go(func() error {
		fmt.Printf("start api server on %s\n", apiServerAddr)
		err := apiServer.ListenAndServe()

		// notify interrupt goroutine shutdown after error
		close(interruptDone)

		fmt.Printf("api server goroutine exit\n")
		return err
	})

	// interrupt goroutine
	g.Go(func() error {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		for {
			select {
			case s := <-c:
				fmt.Printf("program receive signal '%v'\n", s)

				// notify api server shutdown after signal received
				shutdownApiServer()

				fmt.Printf("interrupt goroutine exit\n")
				return Interrupt
			case <-interruptDone:
				// return after channel closed
				fmt.Printf("interrupt goroutine exit\n")
				return nil
			}
		}
	})

	//g.Go(func() error {
	//	mux := http.NewServeMux()
	//	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
	//		fmt.Fprintf(writer, "ok")
	//	})
	//	return http.ListenAndServe(":8081", mux)
	//})

	if err := g.Wait(); err != nil {
		fmt.Printf("errgroup done with error %v\n", err)
	}

	fmt.Printf("program exit\n")
}
