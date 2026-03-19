package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/iceymoss/go-task/pkg/auth"
	"github.com/iceymoss/go-task/pkg/db"
	"github.com/iceymoss/go-task/pkg/db/models"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SESSION_KEY = "user:session:"

	TOKEN_EXPIRE = 24 * time.Hour
)

// AuthService 认证服务
type AuthService struct {
	jwtService  *auth.JWTService
	tokenExpire time.Duration
}

// NewAuthService 创建认证服务
func NewAuthService(jwtService *auth.JWTService, tokenExpire time.Duration) *AuthService {
	return &AuthService{
		jwtService:  jwtService,
		tokenExpire: tokenExpire,
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, username, password string) (*models.User, string, error) {
	dbConn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)
	var user models.User
	result := dbConn.Where("username = ?", username).First(&user)
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
		ID:        uuid.New().String(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(s.tokenExpire),
	}

	rdb := db.GetRedisConn()
	key := SESSION_KEY + strconv.Itoa(int(user.ID))
	if err = rdb.Set(ctx, key, session, TOKEN_EXPIRE).Err(); err != nil {
		return nil, "", err
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	dbConn.Model(&models.User{}).Save(&user)

	return &user, token, nil
}

// Logout 用户登出
func (s *AuthService) Logout(ctx context.Context, token string) error {
	// 解析token
	userInfo, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return err
	}

	uid := userInfo.UserID

	// 删除会话记录
	rdb := db.GetRedisConn()
	if err := rdb.Del(ctx, SESSION_KEY+strconv.Itoa(int(uid))).Err(); err != nil {
		return err
	}
	return nil
}

// ValidateSession 验证会话
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*models.Session, error) {
	// 解析token
	userInfo, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	uid := userInfo.UserID

	rdb := db.GetRedisConn()
	cachedSession := rdb.Get(ctx, SESSION_KEY+strconv.Itoa(int(uid)))
	if cachedSession.Err() != nil {
		if errors.Is(cachedSession.Err(), redis.Nil) {
			return nil, errors.New("session not found")
		}
		return nil, cachedSession.Err()
	}

	cachedSession.Val()

	session := &models.Session{}
	err = json.Unmarshal([]byte(cachedSession.Val()), session)
	if err != nil {
		return nil, err
	}

	// 检查是否过期
	if session.IsExpired() {
		return nil, errors.New("session expired")
	}

	return session, nil
}

// RefreshToken 刷新token
func (s *AuthService) RefreshToken(ctx context.Context, oldToken string) (string, error) {
	// 验证旧token
	session, err := s.ValidateSession(ctx, oldToken)
	if err != nil {
		return "", err
	}

	// 生成新token
	newToken, err := s.jwtService.RefreshToken(oldToken)
	if err != nil {
		return "", err
	}

	// 创建新会话
	newSession := &models.Session{
		ID:        uuid.New().String(),
		UserID:    session.UserID,
		Token:     newToken,
		ExpiresAt: time.Now().Add(s.tokenExpire),
	}

	rdb := db.GetRedisConn()
	key := SESSION_KEY + strconv.Itoa(int(newSession.UserID))
	if err = rdb.Set(ctx, key, session, TOKEN_EXPIRE).Err(); err != nil {
		return "", err
	}

	return newToken, nil
}

// GetCurrentUser 获取当前用户
func (s *AuthService) GetCurrentUser(userID uint) (*models.User, error) {
	dbConn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)
	var user models.User
	result := dbConn.Model(&models.User{}).First(&user, userID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// InitDefaultUser 初始化默认管理员用户
func (s *AuthService) InitDefaultUser(username, password, email string) error {
	dbConn := db.GetMysqlConn(db.MYSQL_DB_GO_TASK)
	// 检查是否已存在管理员
	var count int64
	dbConn.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
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

	return dbConn.Model(&models.User{}).Create(admin).Error
}
