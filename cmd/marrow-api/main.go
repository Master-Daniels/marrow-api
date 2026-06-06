// cmd/marrow-api/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Master-Daniels/marrow/internal/config"
	"github.com/Master-Daniels/marrow/internal/feed"
	"github.com/Master-Daniels/marrow/internal/logger"
	"github.com/Master-Daniels/marrow/internal/storage/sqlite"
	transport "github.com/Master-Daniels/marrow/internal/transport/http"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Env Var load error — continuing with environment", "error: ", err)
	}

	appconfig, err := config.LoadAppConfig()
	if err != nil {
		log.Fatalf("failed to load app config: %v", err)
	}

	logger := logger.NewLogger()

	// 1️⃣ Load configuration (will be watched later)
	configPath := fmt.Sprintf("%s%s", appconfig.ConfigPath, "sources.yml")
	sources, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load sources: %v", err)
	}

	// 2️⃣ Open SQLite DB (file will be created under ./data)
	dbPath := "./data/marrow.db"
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	repo, err := sqlite.New(dbPath)
	if err != nil {
		log.Fatalf("sqlite init failed: %v", err)
	}

	// Create a shared context for all operations
	feedCtx, cancelFeed := context.WithCancel(context.Background())
	defer cancelFeed()

	// 3️⃣ start a poller that reads the sources and for each source starts a goroutine that polls the source and saves items to the database
	feedSrv := feed.New(feedCtx, repo, logger, appconfig)
	feedSrv.StartAll(sources.Sources)

	// 4️⃣ Set up Config Watcher for Hot-Reload
	err = config.Watch(configPath, func() {
		logger.Info("config change detected, reloading sources...")
		newSources, err := config.Load(configPath)
		if err != nil {
			logger.Error("failed to reload config file", "error", err)
			return
		}
		feedSrv.SyncSources(newSources.Sources)
	})
	if err != nil {
		logger.Error("failed to initialize config watcher", "error", err)
		log.Fatalf("config watcher init failed: %v", err)
	}

	// 4️⃣ Build HTTP server
	h := transport.New(transport.NewHandler(repo, feedSrv))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", appconfig.Port),
		Handler: h,
	}

	// 5️⃣ Graceful shutdown handling
	idleConns := make(chan struct{})
	go func() {
		log.Printf("🚀 server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
		close(idleConns)
	}()

	// Wait for SIGINT/SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
	log.Println("🛑 signal received, shutting down…")

	// Ensure all background workers stop gracefully on shutdown
	feedSrv.StopAll()

	// Give active requests a chance to finish
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	<-idleConns
	log.Println("✅ goodbye!")
}
