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
