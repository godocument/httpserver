package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

var Interrupt = errors.New("caught interrupt signal")

func main() {
	ctx := context.Background()
	g, _ := errgroup.WithContext(context.Background())

	// api server
	var apiServer http.Server
	// healthz server
	var healthzServer http.Server

	// interrupt goroutine exit after receive message from this channel
	interruptQuit := make(chan struct{})
	var closeOnce sync.Once

	shutdownApiServer := func() {
		// wait 30s for server shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		err := apiServer.Shutdown(shutdownCtx)
		if err != nil {
			fmt.Printf("fail to shutdown api server: %v\n", err)
		}
	}

	shutdownHealthzServer := func() {
		// wait 30s for server shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		err := healthzServer.Shutdown(shutdownCtx)
		if err != nil {
			fmt.Printf("fail to shutdown api server: %v\n", err)
		}
	}

	// api server goroutine
	{
		apiMux := http.NewServeMux()
		apiMux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			fmt.Fprintf(writer, "hello world")
		})
		apiServerAddr := ":8080"
		apiServer = http.Server{
			Addr:    apiServerAddr,
			Handler: apiMux,
		}

		g.Go(func() error {
			fmt.Printf("start api server on %s\n", apiServerAddr)
			err := apiServer.ListenAndServe()

			// notify healthz server goroutine shutdown
			shutdownHealthzServer()

			// notify interrupt goroutine shutdown
			closeOnce.Do(func() {
				close(interruptQuit)
			})

			fmt.Printf("api server goroutine exit\n")
			return err
		})
	}

	// healthz server goroutine
	{
		healthzMux := http.NewServeMux()
		healthzMux.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
			fmt.Fprintf(writer, "ok")
		})
		healthzServerAddr := ":8081"
		healthzServer = http.Server{
			Addr:    healthzServerAddr,
			Handler: healthzMux,
		}

		g.Go(func() error {
			fmt.Printf("start healthz server on %s\n", healthzServerAddr)
			err := healthzServer.ListenAndServe()

			// notify api server goroutine shutdown
			shutdownApiServer()

			// notify interrupt goroutine shutdown
			closeOnce.Do(func() {
				close(interruptQuit)
			})

			fmt.Printf("healthz server goroutine exit\n")
			return err
		})
	}

	// interrupt goroutine
	{
		g.Go(func() error {
			c := make(chan os.Signal)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			for {
				select {
				case s := <-c:
					fmt.Printf("program receive signal '%v'\n", s)

					// shutdown api server after signal received
					shutdownApiServer()

					// shutdown healthz server after signal received
					shutdownHealthzServer()

					fmt.Printf("interrupt goroutine exit\n")
					return Interrupt
				case <-interruptQuit:
					// return after channel closed
					fmt.Printf("interrupt goroutine exit\n")
					return nil
				}
			}
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Printf("errgroup done with error %v\n", err)
	}

	fmt.Printf("program exit\n")
}
