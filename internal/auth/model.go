package auth

import (
	"encoding/json"
	"time"
)

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Email    string          `json:"email" example:"john.doe@example.com"`
	Password string          `json:"password" example:"securePassword123"`
	Name     json.RawMessage `json:"name" swaggertype:"object"`
	LoginID  *string         `json:"login_id,omitempty" example:"john.doe"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	LoginID  string `json:"login_id" example:"john.doe@example.com"`
	Password string `json:"password" example:"securePassword123"`
}

// TokenResponse represents the access token response
type TokenResponse struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int64  `json:"expires_in" example:"3600"`
}

// RefreshTokenRequest represents the request for token refresh
type RefreshTokenRequest struct {
	// RefreshToken is provided via HTTP-only cookie, not in body
}

// UserInfo represents basic user information in token claims
type UserInfo struct {
	ID      string          `json:"id" example:"01912345-6789-7abc-def0-123456789abc"`
	LoginID string          `json:"login_id" example:"john.doe"`
	Name    json.RawMessage `json:"name" swaggertype:"object"`
	Email   string          `json:"email" example:"john.doe@example.com"`
}

// TokenClaims represents JWT claims
type TokenClaims struct {
	UserID   string   `json:"user_id"`
	LoginID  string   `json:"login_id"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	TokenID  string   `json:"jti"`
	IssuedAt int64    `json:"iat"`
	ExpireAt int64    `json:"exp"`
	Issuer   string   `json:"iss"`
}

// UserToken represents a stored refresh token in the database
type UserToken struct {
	ID                int64
	UserID            int
	TokenHash         string
	ExpiresAt         time.Time
	IsRevoked         bool
	ReplacedByTokenID *int64
	ParentTokenID     *int64
	ClientIP          *string
	UserAgent         *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Group represents a user group
type Group struct {
	ID          int
	PublicID    string
	Name        json.RawMessage
	Description json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Role represents a role for access control
type Role struct {
	ID          int
	Name        string
	Description json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AuthUser represents user data needed for authentication
type AuthUser struct {
	ID           int
	PublicID     string
	LoginID      string
	Email        string
	Name         json.RawMessage
	PasswordHash string
	IsDeleted    bool
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Unauthorized"`
	Message string `json:"message" example:"Invalid credentials"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// RegisterResponse represents the response after successful registration
type RegisterResponse struct {
	User    UserInfo      `json:"user"`
	Tokens  TokenResponse `json:"tokens"`
	Message string        `json:"message" example:"User registered successfully"`
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	User   UserInfo      `json:"user"`
	Tokens TokenResponse `json:"tokens"`
}

// MeResponse represents the current user information response
type MeResponse struct {
	User  UserInfo `json:"user"`
	Roles []string `json:"roles"`
}

// Token configuration constants
const (
	AccessTokenDuration  = 15 * time.Minute      // Access token valid for 15 minutes
	RefreshTokenDuration = 7 * 24 * time.Hour    // Refresh token valid for 7 days
	TokenIssuer          = "knowledgecenter-api"
	PublicGroupID        = "public"              // Default group for new users
)

// Cookie configuration constants
const (
	RefreshTokenCookieName = "refresh_token"
	RefreshTokenCookiePath = "/api/auth"
)
