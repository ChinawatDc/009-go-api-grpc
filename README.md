# go-api-grpc — พัฒนา gRPC API และเชื่อมต่อกับ REST API (Go) แบบละเอียด

บทนี้ทำ “ของจริง” ตั้งแต่ศูนย์:
- ออกแบบ `.proto` ที่มี **service + rpc**
- Generate Go code ด้วย `protoc`
- สร้าง **gRPC Server**
- สร้าง **gRPC Client**
- ทำ **REST API (Gin)** เป็น “Adapter” เรียก gRPC ข้างใน (REST ↔ gRPC bridge)
- ใส่แนวทาง Production: config, graceful shutdown, timeouts, interceptors, error mapping

> หมายเหตุ: บทนี้ “เชื่อม REST ด้วย Gin ที่เรียก gRPC client” (ง่ายและตรง)  
> ถ้าจะ expose REST จาก proto โดยตรงแบบมาตรฐานอุตสาหกรรม ให้ไปต่อบท `go-api-grpc-gateway`

---

## 0) Prerequisites

### 0.1 ติดตั้ง `protoc`
ตรวจสอบ:
```bash
protoc --version
```

Ubuntu/Debian/WSL:
```bash
sudo apt update
sudo apt install -y protobuf-compiler
```

### 0.2 ติดตั้ง Go plugins สำหรับ proto + grpc
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

เพิ่ม PATH (ถ้ารันแล้วหา plugin ไม่เจอ):
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

ตรวจสอบ:
```bash
protoc-gen-go --version
protoc-gen-go-grpc --version
```

---

## 1) Create Project Structure

```bash
mkdir -p go-api-grpc
cd go-api-grpc

mkdir -p proto/user/v1
mkdir -p gen/go
mkdir -p internal/config
mkdir -p internal/user
mkdir -p internal/transport/grpcserver
mkdir -p internal/transport/rest
mkdir -p cmd/grpc-server
mkdir -p cmd/rest-api
```

โครงสร้าง:
```text
go-api-grpc/
  proto/
    user/
      v1/
        user.proto
  gen/
    go/
      (generated)
  internal/
    config/
      config.go
    user/
      service.go
      store.go
    transport/
      grpcserver/
        server.go
        handler.go
      rest/
        http.go
        handler.go
  cmd/
    grpc-server/
      main.go
    rest-api/
      main.go
  Makefile
  go.mod
  README.md
```

แนวคิด production:
- `proto/` = source of truth
- `gen/` = generated code (ห้ามแก้มือ)
- `internal/` = business logic + transports แยกชัด
- `cmd/` = executable แยก service

---

## 2) Initialize Go Module

แก้ module path ให้ตรง repo คุณ:
```bash
go mod init github.com/ChinawatDc/go-api-grpc
```

ติดตั้ง deps:
```bash
go get google.golang.org/grpc@latest
go get google.golang.org/protobuf@latest
go get github.com/gin-gonic/gin@latest
go get github.com/google/uuid@latest
```

---

## 3) Define Proto (Service + Messages)

สร้างไฟล์ `proto/user/v1/user.proto`

```proto
syntax = "proto3";

package user.v1;

option go_package = "github.com/ChinawatDc/go-api-grpc/gen/go/user/v1;userv1";

message User {
  string id = 1;
  string email = 2;
  string name = 3;
}

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
}

message CreateUserRequest {
  string email = 1;
  string name = 2;
}

message CreateUserResponse {
  User user = 1;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
}
```

Best practices:
- มี `v1` ตั้งแต่แรก
- `go_package` ชี้ path จริงใน repo
- Field number ห้ามเปลี่ยนหลังปล่อยใช้งาน

---

## 4) Generate Code

คำสั่ง:
```bash
protoc -I proto --go_out=gen/go --go_opt=paths=source_relative --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative proto/user/v1/user.proto
```

ผลลัพธ์:
```text
gen/go/user/v1/user.pb.go
gen/go/user/v1/user_grpc.pb.go
```

---

## 5) Makefile

สร้าง `Makefile`:

```makefile
.PHONY: proto tidy run-grpc run-rest

PROTO_DIR=proto
GEN_DIR=gen/go

proto:
	@echo ">> generating protobuf..."
	protoc -I $(PROTO_DIR) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/user/v1/user.proto
	@echo ">> done"

tidy:
	go mod tidy

run-grpc:
	go run ./cmd/grpc-server

run-rest:
	go run ./cmd/rest-api
```

---

## 6) Business Logic

`internal/user/store.go`
```go
package user

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("user not found")

type Store interface {
	Get(id string) (User, error)
	Create(u User) (User, error)
}

type InMemoryStore struct {
	mu    sync.RWMutex
	items map[string]User
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: make(map[string]User)}
}

func (s *InMemoryStore) Get(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.items[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (s *InMemoryStore) Create(u User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[u.ID] = u
	return u, nil
}
```

`internal/user/service.go`
```go
package user

import "github.com/google/uuid"

type User struct {
	ID    string
	Email string
	Name  string
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetUser(id string) (User, error) {
	return s.store.Get(id)
}

func (s *Service) CreateUser(email, name string) (User, error) {
	u := User{
		ID:    uuid.NewString(),
		Email: email,
		Name:  name,
	}
	return s.store.Create(u)
}
```

---

## 7) gRPC Server

`internal/transport/grpcserver/handler.go`
```go
package grpcserver

import (
	"context"

	userv1 "github.com/ChinawatDc/go-api-grpc/gen/go/user/v1"
	"github.com/ChinawatDc/go-api-grpc/internal/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	userv1.UnimplementedUserServiceServer
	svc *user.Service
}

func NewHandler(svc *user.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	u, err := h.svc.GetUser(req.GetId())
	if err != nil {
		if err == user.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &userv1.GetUserResponse{
		User: &userv1.User{Id: u.ID, Email: u.Email, Name: u.Name},
	}, nil
}

func (h *Handler) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	if req.GetEmail() == "" || req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and name are required")
	}
	u, err := h.svc.CreateUser(req.GetEmail(), req.GetName())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &userv1.CreateUserResponse{
		User: &userv1.User{Id: u.ID, Email: u.Email, Name: u.Name},
	}, nil
}
```

`internal/transport/grpcserver/server.go`
```go
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
```

`cmd/grpc-server/main.go`
```go
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
```

---

## 8) REST API (Gin) เรียก gRPC Client

`internal/transport/rest/http.go`
```go
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
```

`internal/transport/rest/handler.go`
```go
package rest

import (
	"context"
	"net/http"
	"time"

	userv1 "github.com/ChinawatDc/go-api-grpc/gen/go/user/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RegisterRoutes(s *Server) {
	api := s.Engine.Group("/api/v1")
	{
		api.GET("/users/:id", s.getUser)
		api.POST("/users", s.createUser)
	}
}

func (s *Server) getUser(c *gin.Context) {
	client := userv1.NewUserServiceClient(s.GRPCConn)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	resp, err := client.GetUser(ctx, &userv1.GetUserRequest{Id: c.Param("id")})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":    resp.GetUser().GetId(),
		"email": resp.GetUser().GetEmail(),
		"name":  resp.GetUser().GetName(),
	})
}

type createUserBody struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
}

func (s *Server) createUser(c *gin.Context) {
	var body createUserBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client := userv1.NewUserServiceClient(s.GRPCConn)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	resp, err := client.CreateUser(ctx, &userv1.CreateUserRequest{
		Email: body.Email,
		Name:  body.Name,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":    resp.GetUser().GetId(),
		"email": resp.GetUser().GetEmail(),
		"name":  resp.GetUser().GetName(),
	})
}

func writeGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unknown error"})
		return
	}

	switch st.Code() {
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
	case codes.DeadlineExceeded:
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "upstream timeout"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
	}
}
```

`cmd/rest-api/main.go`
```go
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
```

---

## 9) Run & Test

Generate:
```bash
make proto
make tidy
```

Run gRPC:
```bash
make run-grpc
```

Run REST:
```bash
make run-rest
```

Test REST:
```bash
curl -X POST http://localhost:8080/api/v1/users   -H 'Content-Type: application/json'   -d '{"email":"dev@example.com","name":"Dev One"}'
```

แล้วเอา `id` ไปยิง:
```bash
curl http://localhost:8080/api/v1/users/<ID>
```

---

## 10) Production Notes

- **Timeouts** ทุก hop (REST -> gRPC -> DB)
- gRPC ใช้ `codes.*` แล้ว REST map เป็น HTTP status ให้ถูก
- Production ควรใช้ **TLS/mTLS** แทน `WithInsecure()`
- ใส่ Interceptors สำหรับ logging/metrics/auth/recovery
- Breaking change ให้ทำ `v2` (อย่าแอบแก้ `v1`)

---

## Next Lesson
ไปต่อ `go-api-grpc-gateway` เพื่อ generate REST endpoints จาก proto โดยตรงแบบมาตรฐาน

---

MIT License
# 009-go-api-grpc
