package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"reverse-proxy/internal/proxy"
	"reverse-proxy/pkg/logger"
	"syscall"
	"time"
)

func main() {
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
	}

	handler := proxy.NewReverseProxy(backends)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Reverse proxy started on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server:", err)
		}
	}()

	<-quit
	logger.Info("Shutdown signal received, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}

	logger.Info("Server exited gracefully")
}
