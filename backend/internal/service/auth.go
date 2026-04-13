package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/dukedhal/taskflow/internal/model"
	"github.com/dukedhal/taskflow/internal/repository"
)

// Claims are the JWT payload fields we embed in every token.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type AuthService struct {
	users     *repository.UserRepository
	jwtSecret []byte
}

func NewAuthService(users *repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		users:     users,
		jwtSecret: []byte(jwtSecret),
	}
}

// RegisterInput is the validated payload from POST /auth/register.
type RegisterInput struct {
	Name     string `json:"name"     validate:"required,min=1,max=255"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginInput is the validated payload from POST /auth/login.
type LoginInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthResponse is returned from both register and login.
type AuthResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// Register creates a new user and returns a JWT.
// Returns ErrConflict if the email is already taken.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &model.User{
		Name:     strings.TrimSpace(in.Name),
		Email:    strings.ToLower(strings.TrimSpace(in.Email)),
		Password: string(hash),
	}

	if err := s.users.Create(ctx, u); err != nil {
		// Postgres unique violation on email → translate to domain error.
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, model.ErrConflict
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.generateToken(u)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: u}, nil
}

// Login verifies credentials and returns a JWT.
// Returns ErrUnauthorized for both "not found" and "wrong password" — never
// distinguish them to the caller (prevents user enumeration).
func (s *AuthService) Login(ctx context.Context, in LoginInput) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	u, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.ErrUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)); err != nil {
		return nil, model.ErrUnauthorized
	}

	token, err := s.generateToken(u)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: u}, nil
}

// ValidateToken parses and verifies a JWT, returning its claims on success.
func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, model.ErrUnauthorized
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, model.ErrUnauthorized
	}

	return claims, nil
}

func (s *AuthService) generateToken(u *model.User) (string, error) {
	claims := Claims{
		UserID: u.ID,
		Email:  u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
