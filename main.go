package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hohn/mrvacommander/pkg/agent"
	"github.com/hohn/mrvacommander/pkg/deploy"
)

func startHealthEndpoint() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	port := os.Getenv("AGENT_PORT")
	if port == "" {
		port = "8081"
	}

	slog.Info("Starting health endpoint", "port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		slog.Error("Error starting health endpoint", "error", err)
	}
}

func main() {
	slog.Info("Starting agent")
	workerCount := flag.Int("workers", 0, "number of workers")
	logLevel := flag.String("loglevel", "debug", "Set log level: debug, info, warn, error")
	flag.Parse()

	// Apply 'loglevel' flag
	switch *logLevel {
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	default:
		log.Printf("Invalid logging verbosity level: %s", *logLevel)
		os.Exit(1)
	}

	isAgent := true

	rabbitMQQueue, err := deploy.InitRabbitMQ(isAgent)
	if err != nil {
		slog.Error("Failed to initialize RabbitMQ", slog.Any("error", err))
		os.Exit(1)
	}
	defer rabbitMQQueue.Close()

	artifacts, err := deploy.InitMinIOArtifactStore()
	if err != nil {
		slog.Error("Failed to initialize artifact store", slog.Any("error", err))
		os.Exit(1)
	}

	// XX: deprecate
	// databases, err := deploy.InitMinIOCodeQLDatabaseStore()
	// if err != nil {
	// 	slog.Error("Failed to initialize database store", slog.Any("error", err))
	// 	os.Exit(1)
	// }

	databases, err := deploy.InitHEPCDatabaseStore()
	if err != nil {
		slog.Error("Failed to initialize database store", slog.Any("error", err))
		os.Exit(1)
	}

	serverState := deploy.InitPGState()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	go agent.StartAndMonitorWorkers(ctx, artifacts, databases, rabbitMQQueue, serverState, *workerCount, &wg)
	slog.Info("Agent started")

	// Start health check endpoint
	go startHealthEndpoint()

	// Gracefully exit on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Shutting down agent")
	cancel()
	wg.Wait()
	slog.Info("Agent shutdown complete")
}
