package auth

import (
	"errors"
	"time"

	"github.com/iceymoss/go-task/pkg/db/models"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	db          *gorm.DB
	jwtService  *JWTService
	tokenExpire time.Duration
}

// NewAuthService 创建认证服务
func NewAuthService(db *gorm.DB, jwtService *JWTService, tokenExpire time.Duration) *AuthService {
	return &AuthService{
		db:          db,
		jwtService:  jwtService,
		tokenExpire: tokenExpire,
	}
}

// Login 用户登录
func (s *AuthService) Login(username, password string) (*models.User, string, error) {
	// 查找用户
	var user models.User
	result := s.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("invalid username or password")
		}
		return nil, "", result.Error
	}

	// 检查用户是否激活
	if !user.IsActive {
		return nil, "", errors.New("user account is disabled")
	}

	// 验证密码
	if err := user.CheckPassword(password); err != nil {
		return nil, "", errors.New("invalid username or password")
	}

	// 生成JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, "", err
	}

	// 创建会话记录
	session := &models.Session{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(s.tokenExpire),
	}
	if err := s.db.Create(session).Error; err != nil {
		return nil, "", err
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Save(&user)

	return &user, token, nil
}

// Logout 用户登出
func (s *AuthService) Logout(token string) error {
	// 删除会话记录
	return s.db.Where("token = ?", token).Delete(&models.Session{}).Error
}

// ValidateSession 验证会话
func (s *AuthService) ValidateSession(token string) (*models.Session, error) {
	var session models.Session
	result := s.db.Where("token = ?", token).First(&session)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("session not found")
		}
		return nil, result.Error
	}

	// 检查是否过期
	if session.IsExpired() {
		return nil, errors.New("session expired")
	}

	return &session, nil
}

// RefreshToken 刷新token
func (s *AuthService) RefreshToken(oldToken string) (string, error) {
	// 验证旧token
	session, err := s.ValidateSession(oldToken)
	if err != nil {
		return "", err
	}

	// 获取用户信息
	var user models.User
	if err := s.db.First(&user, session.UserID).Error; err != nil {
		return "", err
	}

	// 生成新token
	newToken, err := s.jwtService.RefreshToken(oldToken)
	if err != nil {
		return "", err
	}

	// 创建新会话
	newSession := &models.Session{
		UserID:    user.ID,
		Token:     newToken,
		ExpiresAt: time.Now().Add(s.tokenExpire),
	}
	if err := s.db.Create(newSession).Error; err != nil {
		return "", err
	}

	// 删除旧会话
	s.db.Delete(session)

	return newToken, nil
}

// GetCurrentUser 获取当前用户
func (s *AuthService) GetCurrentUser(userID uint) (*models.User, error) {
	var user models.User
	result := s.db.First(&user, userID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// InitDefaultUser 初始化默认管理员用户
func (s *AuthService) InitDefaultUser(username, password, email string) error {
	// 检查是否已存在管理员
	var count int64
	s.db.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return nil // 已存在管理员，不创建
	}

	// 创建默认管理员
	admin := &models.User{
		Username: username,
		Email:    email,
		Role:     "admin",
		IsActive: true,
	}

	if err := admin.SetPassword(password); err != nil {
		return err
	}

	return s.db.Create(admin).Error
}