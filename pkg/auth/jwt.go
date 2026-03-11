package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTService JWT服务
type JWTService struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewJWTService 创建JWT服务
func NewJWTService(secretKey string, duration time.Duration) *JWTService {
	return &JWTService{
		secretKey:     secretKey,
		tokenDuration: duration,
	}
}

// GenerateToken 生成JWT token
func (s *JWTService) GenerateToken(userID uint, username, role string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "go-task",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// ValidateToken 验证JWT token
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken 刷新token
func (s *JWTService) RefreshToken(oldTokenString string) (string, error) {
	claims, err := s.ValidateToken(oldTokenString)
	if err != nil {
		return "", err
	}

	// 生成新token
	return s.GenerateToken(claims.UserID, claims.Username, claims.Role)
}