package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ramhandean/diorama-ops/internal/admin"
	"github.com/ramhandean/diorama-ops/internal/config"
	"github.com/ramhandean/diorama-ops/internal/db"
	"github.com/ramhandean/diorama-ops/internal/server"
	"github.com/ramhandean/diorama-ops/internal/tenants"
	_ "modernc.org/sqlite"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "serve":
		runServe()
	case "healthcheck":
		runHealthcheck()
	case "version", "--version", "-v":
		fmt.Printf("dioramaops %s\n", version)
	case "import":
		runImport()
	case "export":
		runExport()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: dioramaops <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  serve        Start the DioramaOps telemetry hub and dashboard server")
	fmt.Println("  healthcheck  Probe local server health endpoint for container healthcheck")
	fmt.Println("  version      Print version information")
	fmt.Println("  import       Import tenants from JSON file into SQLite database")
	fmt.Println("  export       Export tenants from SQLite database to JSON file")
}

func runServe() {
	// Structured JSON logging, no raw IPs logged
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server fatal error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("DioramaOps hub ready", "version", version, "port", cfg.Port)

	<-stop
	slog.Info("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("error during server shutdown", "error", err)
	}
	slog.Info("server stopped")
}

func runHealthcheck() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "7437"
	}
	url := fmt.Sprintf("http://127.0.0.1:%s/healthz", port)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck request build failed: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck status non-200: %d\n", resp.StatusCode)
		os.Exit(1)
	}
	os.Exit(0)
}

func runImport() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: dioramaops import <file.json>")
		os.Exit(1)
	}
	filePath := os.Args[2]
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open %s: %v\n", filePath, err)
		os.Exit(1)
	}
	defer file.Close()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	dbConn, err := db.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	store := tenants.NewStore(dbConn)
	sm := admin.NewSyncManager(store)

	count, err := sm.Import(context.Background(), file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "import error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully imported %d tenant(s) from %s\n", count, filePath)
}

func runExport() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	dbConn, err := db.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	store := tenants.NewStore(dbConn)
	sm := admin.NewSyncManager(store)

	if len(os.Args) >= 3 {
		filePath := os.Args[2]
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create %s: %v\n", filePath, err)
			os.Exit(1)
		}
		defer file.Close()
		if err := sm.Export(context.Background(), file); err != nil {
			fmt.Fprintf(os.Stderr, "export error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully exported tenants to %s\n", filePath)
	} else {
		_ = sm.Export(context.Background(), os.Stdout)
	}
}
