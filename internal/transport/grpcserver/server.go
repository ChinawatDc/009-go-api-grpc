package grpcserver

import (
	"context"
	"fmt"
	"net"
	"time"

	userv1 "github.com/ChinawatDc/go-api-grpc/gen/go/user/v1"
	"google.golang.org/grpc"
)

type Server struct {
	grpcServer *grpc.Server
	addr       string
}

func New(addr string, handler userv1.UserServiceServer, opts ...grpc.ServerOption) *Server {
	s := grpc.NewServer(opts...)
	userv1.RegisterUserServiceServer(s, handler)
	return &Server{grpcServer: s, addr: addr}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop(ctx context.Context) error {
	ch := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(ch)
	}()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		return ctx.Err()
	}
}

func DefaultStopContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
