package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChinawatDc/go-api-grpc/internal/transport/rest"
)

func main() {
	cfg := rest.Config{
		HTTPAddr: ":8080",
		GRPCAddr: "localhost:50051",
	}

	s, err := rest.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer s.GRPCConn.Close()

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      s.Engine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("REST API listening on", cfg.HTTPAddr, "-> gRPC", cfg.GRPCAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("REST API stopped")
}
