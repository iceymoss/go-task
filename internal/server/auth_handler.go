package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iceymoss/go-task/internal/conf"
	"github.com/iceymoss/go-task/pkg/auth"
	"github.com/iceymoss/go-task/pkg/db/models"
	"gorm.io/gorm"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *auth.AuthService
	jwtService  *auth.JWTService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(db *gorm.DB, cfg *conf.Config) *AuthHandler {
	jwtService := auth.NewJWTService(cfg.Auth.JWTSecret, time.Duration(cfg.Auth.TokenExpireHrs)*time.Hour)
	authService := auth.NewAuthService(db, jwtService, time.Duration(cfg.Auth.TokenExpireHrs)*time.Hour)

	// 初始化默认管理员
	if err := authService.InitDefaultUser(
		cfg.Auth.DefaultAdmin.Username,
		cfg.Auth.DefaultAdmin.Password,
		cfg.Auth.DefaultAdmin.Email,
	); err != nil {
		// 这里只记录日志，不影响服务启动
		// log.Printf("⚠️ Failed to init default admin: %v", err)
	}

	return &AuthHandler{
		authService: authService,
		jwtService:  jwtService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 隐藏密码哈希
	user.PasswordHash = ""

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}

	// 移除Bearer前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if err := h.authService.Logout(token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// GetMe 获取当前用户信息
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	user, err := h.authService.GetCurrentUser(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 隐藏密码哈希
	user.PasswordHash = ""

	c.JSON(http.StatusOK, user)
}

// RefreshTokenRequest 刷新token请求
type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// RefreshToken 刷新token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newToken, err := h.authService.RefreshToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newToken})
}
