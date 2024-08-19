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
	srv := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Request received")
			resp, err := http.Get("http://127.0.0.1:9999")
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			defer resp.Body.Close()
			io.Copy(w, resp.Body)
		}),
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

	slog.Info("Starting server")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
		os.Exit(1)
	}
	slog.Info("Server closed successfully")
}
