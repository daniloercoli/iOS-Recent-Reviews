package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"backend/internal"
)

func loadConfig() (*internal.Config, error) {
	path := filepath.Join("config", "apps.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return internal.ParseConfig(f)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := os.MkdirAll(filepath.Join("data", "reviews"), 0o755); err != nil {
		log.Fatalf("creating data/reviews: %v", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	st, err := internal.NewFileStore("data")
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	mgr := internal.NewManager(cfg, st)
	mgr.Start()

	mux := internal.BuildMux(cfg, st, mgr)
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      internal.WithCORS(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("HTTP server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for stop signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	// 1) Stop the poller
	mgr.Stop()

	// 2) Close the HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	} else {
		log.Println("server stopped cleanly")
	}
}
