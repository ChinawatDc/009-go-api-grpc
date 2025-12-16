package rest

import (
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type Server struct {
	Engine   *gin.Engine
	GRPCConn *grpc.ClientConn
}

type Config struct {
	HTTPAddr string
	GRPCAddr string
}

func New(cfg Config) (*Server, error) {
	r := gin.New()
	r.Use(gin.Recovery())

	conn, err := grpc.Dial(
		cfg.GRPCAddr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(3*time.Second),
	)
	if err != nil {
		return nil, err
	}

	s := &Server{Engine: r, GRPCConn: conn}
	RegisterRoutes(s)
	return s, nil
}