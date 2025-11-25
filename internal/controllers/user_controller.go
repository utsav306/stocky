package controllers

import (
	"net/http"
	"stockBackend/internal/models"
	"stockBackend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	userRepo repository.UserRepository
	log      *logrus.Logger
}

// NewUserController creates a new user controller
func NewUserController(userRepo repository.UserRepository, log *logrus.Logger) *UserController {
	return &UserController{
		userRepo: userRepo,
		log:      log,
	}
}

// CreateUser creates a new user in the system
// POST /api/v1/users
func (uc *UserController) CreateUser(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Name   string `json:"name" binding:"required"`
		Email  string `json:"email" binding:"required,email"`
	}

	// Parse and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Check if user already exists
	exists, err := uc.userRepo.Exists(c.Request.Context(), req.UserID)
	if err != nil {
		uc.log.Errorf("Failed to check user existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check user existence",
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{
			"error": "User already exists with this user_id",
		})
		return
	}

	// Create user
	user := &models.User{
		UserID: req.UserID,
		Name:   req.Name,
		Email:  req.Email,
	}

	if err := uc.userRepo.Create(c.Request.Context(), user); err != nil {
		uc.log.Errorf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	uc.log.Infof("User created successfully: %s", user.UserID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"data": gin.H{
			"id":         user.ID,
			"user_id":    user.UserID,
			"name":       user.Name,
			"email":      user.Email,
			"created_at": user.CreatedAt,
		},
	})
}

// GetUser retrieves a user by user_id
// GET /api/v1/users/:userId
func (uc *UserController) GetUser(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	user, err := uc.userRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		uc.log.Errorf("Failed to get user: %v", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":         user.ID,
			"user_id":    user.UserID,
			"name":       user.Name,
			"email":      user.Email,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

// ListUsers retrieves all users with pagination
// GET /api/v1/users?limit=10&offset=0
func (uc *UserController) ListUsers(c *gin.Context) {
	// Parse pagination params
	limit := 10
	offset := 0
	
	if l := c.Query("limit"); l != "" {
		if parsed, err := parseIntParam(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	
	if o := c.Query("offset"); o != "" {
		if parsed, err := parseIntParam(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	users, err := uc.userRepo.List(c.Request.Context(), limit, offset)
	if err != nil {
		uc.log.Errorf("Failed to list users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   users,
		"count":  len(users),
		"limit":  limit,
		"offset": offset,
	})
}
