package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Readyness endpoint called", slog.String("host", r.Host))
	})
	mux.HandleFunc("GET /remote", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Remote endpoint called", slog.String("host", r.Host))
		resp, err := http.Get("http://127.0.0.1:9999")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer resp.Body.Close()
		io.Copy(w, resp.Body)
	})

	srv := &http.Server{
		Addr:    os.Getenv("ITR_TEST_ADDR"),
		Handler: mux,
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		slog.Info("Closing server")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
	}()

	slog.Info("Starting server", slog.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
		os.Exit(1)
	}
	slog.Info("Server closed successfully")
}
