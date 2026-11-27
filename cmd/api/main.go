// Command api starts the Devices HTTP API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guilhermequitto/devices-crud/internal/device"
	"github.com/guilhermequitto/devices-crud/internal/storage/sqlite"
	httpapi "github.com/guilhermequitto/devices-crud/internal/transport/http"
)

func main() {
	port := envOr("PORT", "8080")
	dbPath := envOr("DB_PATH", "devices.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	log.Printf("database ready at %s", dbPath)

	svc := device.NewService(sqlite.NewRepository(db))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewRouter(svc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
