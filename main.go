package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"fast-round-api/config"
	"fast-round-api/handlers"
	"fast-round-api/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if *healthcheck {
		runHealthcheck(cfg.Port)
		return
	}

	gin.SetMode(cfg.GinMode)

	store := storage.NewRedisStore(storage.RedisOptions{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("redis close failed: %v", err)
		}
	}()

	if err := store.Ping(context.Background()); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	handler := &handlers.EventHandler{
		Store:  store,
		APIKey: cfg.APIKey,
	}

	r := gin.Default()
	r.Use(handlers.SecurityHeaders)
	r.Use(handlers.LimitBodySize(cfg.MaxBodyBytes))
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}
	r.GET("/health", handlers.HandleHealth)

	v1 := r.Group("/api/v1")
	{
		protected := v1.Group("")
		protected.Use(handler.RequireAPIKey)
		protected.POST("/event", handler.HandleEvent)
		v1.GET("/matches/:match_id", handler.HandleGetMatch)
	}

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		log.Printf("server started on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
}

func runHealthcheck(port string) {
	resp, err := http.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Fatalf("healthcheck failed: %s", resp.Status)
	}
}
