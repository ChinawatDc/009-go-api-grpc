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