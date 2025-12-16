package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChinawatDc/go-api-grpc/internal/transport/grpcserver"
	"github.com/ChinawatDc/go-api-grpc/internal/user"
)

func main() {
	addr := ":50051"

	store := user.NewInMemoryStore()
	svc := user.NewService(store)
	handler := grpcserver.NewHandler(svc)

	srv := grpcserver.New(addr, handler)

	go func() {
		log.Println("gRPC server listening on", addr)
		if err := srv.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	ctx, cancel := grpcserver.DefaultStopContext()
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		log.Println("graceful stop error:", err)
	}
	log.Println("gRPC server stopped")
}