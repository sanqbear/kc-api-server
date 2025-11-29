package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// MockService is a mock implementation of the Service interface for testing
type MockService struct {
	RegisterFunc            func(ctx context.Context, req *RegisterRequest, clientIP, userAgent string) (*RegisterResponse, string, error)
	LoginFunc               func(ctx context.Context, req *LoginRequest, clientIP, userAgent string) (*LoginResponse, string, error)
	RefreshFunc             func(ctx context.Context, refreshToken, clientIP, userAgent string) (*TokenResponse, string, error)
	LogoutFunc              func(ctx context.Context, refreshToken string) error
	LogoutAllFunc           func(ctx context.Context, userID string) error
	GetMeFunc               func(ctx context.Context, userID string) (*MeResponse, error)
	ValidateAccessTokenFunc func(tokenString string) (*TokenClaims, error)
}

func (m *MockService) Register(ctx context.Context, req *RegisterRequest, clientIP, userAgent string) (*RegisterResponse, string, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req, clientIP, userAgent)
	}
	return nil, "", nil
}

func (m *MockService) Login(ctx context.Context, req *LoginRequest, clientIP, userAgent string) (*LoginResponse, string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, req, clientIP, userAgent)
	}
	return nil, "", nil
}

func (m *MockService) Refresh(ctx context.Context, refreshToken, clientIP, userAgent string) (*TokenResponse, string, error) {
	if m.RefreshFunc != nil {
		return m.RefreshFunc(ctx, refreshToken, clientIP, userAgent)
	}
	return nil, "", nil
}

func (m *MockService) Logout(ctx context.Context, refreshToken string) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, refreshToken)
	}
	return nil
}

func (m *MockService) LogoutAll(ctx context.Context, userID string) error {
	if m.LogoutAllFunc != nil {
		return m.LogoutAllFunc(ctx, userID)
	}
	return nil
}

func (m *MockService) GetMe(ctx context.Context, userID string) (*MeResponse, error) {
	if m.GetMeFunc != nil {
		return m.GetMeFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	if m.ValidateAccessTokenFunc != nil {
		return m.ValidateAccessTokenFunc(tokenString)
	}
	return nil, nil
}

func TestHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *RegisterResponse
		mockRefresh    string
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful registration",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Name:     json.RawMessage(`{"en-US": "Test User"}`),
			},
			mockReturn: &RegisterResponse{
				User: UserInfo{
					ID:      "01912345-6789-7abc-def0-123456789abc",
					LoginID: "test@example.com",
					Name:    json.RawMessage(`{"en-US": "Test User"}`),
					Email:   "test@example.com",
				},
				Tokens: TokenResponse{
					AccessToken: "test-access-token",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
				Message: "User registered successfully",
			},
			mockRefresh:    "test-refresh-token",
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid email",
			requestBody: RegisterRequest{
				Email:    "invalid-email",
				Password: "password123",
				Name:     json.RawMessage(`{"en-US": "Test User"}`),
			},
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "email already exists",
			requestBody: RegisterRequest{
				Email:    "existing@example.com",
				Password: "password123",
				Name:     json.RawMessage(`{"en-US": "Test User"}`),
			},
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      ErrEmailExists,
			expectedStatus: http.StatusConflict,
		},
		{
			name: "password too short",
			requestBody: RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
				Name:     json.RawMessage(`{"en-US": "Test User"}`),
			},
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      ErrInvalidPassword,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				RegisterFunc: func(ctx context.Context, req *RegisterRequest, clientIP, userAgent string) (*RegisterResponse, string, error) {
					return tt.mockReturn, tt.mockRefresh, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Check for refresh token cookie on successful registration
			if tt.expectedStatus == http.StatusCreated {
				cookies := rec.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == RefreshTokenCookieName {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected refresh token cookie to be set")
				}
			}
		})
	}
}

func TestHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *LoginResponse
		mockRefresh    string
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful login",
			requestBody: LoginRequest{
				LoginID:  "test@example.com",
				Password: "password123",
			},
			mockReturn: &LoginResponse{
				User: UserInfo{
					ID:      "01912345-6789-7abc-def0-123456789abc",
					LoginID: "test@example.com",
					Name:    json.RawMessage(`{"en-US": "Test User"}`),
					Email:   "test@example.com",
				},
				Tokens: TokenResponse{
					AccessToken: "test-access-token",
					TokenType:   "Bearer",
					ExpiresIn:   900,
				},
			},
			mockRefresh:    "test-refresh-token",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid credentials",
			requestBody: LoginRequest{
				LoginID:  "test@example.com",
				Password: "wrongpassword",
			},
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      ErrInvalidCredentials,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid request body",
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				LoginFunc: func(ctx context.Context, req *LoginRequest, clientIP, userAgent string) (*LoginResponse, string, error) {
					return tt.mockReturn, tt.mockRefresh, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_Refresh(t *testing.T) {
	tests := []struct {
		name           string
		cookie         *http.Cookie
		mockReturn     *TokenResponse
		mockRefresh    string
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful refresh",
			cookie: &http.Cookie{
				Name:  RefreshTokenCookieName,
				Value: "valid-refresh-token",
			},
			mockReturn: &TokenResponse{
				AccessToken: "new-access-token",
				TokenType:   "Bearer",
				ExpiresIn:   900,
			},
			mockRefresh:    "new-refresh-token",
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no cookie",
			cookie:         nil,
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid token",
			cookie: &http.Cookie{
				Name:  RefreshTokenCookieName,
				Value: "invalid-token",
			},
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      ErrInvalidToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "revoked token",
			cookie: &http.Cookie{
				Name:  RefreshTokenCookieName,
				Value: "revoked-token",
			},
			mockReturn:     nil,
			mockRefresh:    "",
			mockError:      ErrTokenRevoked,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				RefreshFunc: func(ctx context.Context, refreshToken, clientIP, userAgent string) (*TokenResponse, string, error) {
					return tt.mockReturn, tt.mockRefresh, tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandler_Logout(t *testing.T) {
	tests := []struct {
		name           string
		cookie         *http.Cookie
		mockError      error
		expectedStatus int
	}{
		{
			name: "successful logout with cookie",
			cookie: &http.Cookie{
				Name:  RefreshTokenCookieName,
				Value: "valid-token",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "logout without cookie",
			cookie:         nil,
			mockError:      nil,
			expectedStatus: http.StatusOK, // Should still succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				LogoutFunc: func(ctx context.Context, refreshToken string) error {
					return tt.mockError
				},
			}

			handler := NewHandler(mockService)
			r := chi.NewRouter()
			handler.RegisterRoutes(r)

			req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Check that cookie is cleared
			cookies := rec.Result().Cookies()
			for _, cookie := range cookies {
				if cookie.Name == RefreshTokenCookieName {
					if cookie.MaxAge != -1 {
						t.Error("expected cookie to be cleared (MaxAge should be -1)")
					}
				}
			}
		})
	}
}

func TestMiddleware_Authenticate(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		mockClaims     *TokenClaims
		mockError      error
		expectedStatus int
	}{
		{
			name:       "valid token",
			authHeader: "Bearer valid-token",
			mockClaims: &TokenClaims{
				UserID:  "user-123",
				LoginID: "test@example.com",
				Email:   "test@example.com",
				Roles:   []string{"user"},
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			mockClaims:     nil,
			mockError:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid header format",
			authHeader:     "InvalidFormat",
			mockClaims:     nil,
			mockError:      nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			mockClaims:     nil,
			mockError:      ErrInvalidToken,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{
				ValidateAccessTokenFunc: func(tokenString string) (*TokenClaims, error) {
					return tt.mockClaims, tt.mockError
				},
			}

			middleware := NewMiddleware(mockService)

			// Create a test handler that just returns 200 OK
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			middleware.Authenticate(testHandler).ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestMiddleware_RequireRoles(t *testing.T) {
	tests := []struct {
		name           string
		userRoles      []string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "user has required role",
			userRoles:      []string{"admin", "user"},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user has one of required roles",
			userRoles:      []string{"user"},
			requiredRoles:  []string{"admin", "user"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user missing required role",
			userRoles:      []string{"user"},
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no user roles",
			userRoles:      nil,
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockService{}
			middleware := NewMiddleware(mockService)

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Add roles to context
			ctx := context.WithValue(req.Context(), userRolesKey, tt.userRoles)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			middleware.RequireRoles(tt.requiredRoles...)(testHandler).ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestContextHelpers(t *testing.T) {
	t.Run("GetUserIDFromContext", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, "test-user-id")
		userID := GetUserIDFromContext(ctx)
		if userID != "test-user-id" {
			t.Errorf("expected 'test-user-id', got '%s'", userID)
		}

		// Test with empty context
		emptyUserID := GetUserIDFromContext(context.Background())
		if emptyUserID != "" {
			t.Errorf("expected empty string, got '%s'", emptyUserID)
		}
	})

	t.Run("GetUserRolesFromContext", func(t *testing.T) {
		roles := []string{"admin", "user"}
		ctx := context.WithValue(context.Background(), userRolesKey, roles)
		result := GetUserRolesFromContext(ctx)
		if len(result) != 2 {
			t.Errorf("expected 2 roles, got %d", len(result))
		}
	})

	t.Run("IsAuthenticated", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey, "test-user-id")
		if !IsAuthenticated(ctx) {
			t.Error("expected IsAuthenticated to return true")
		}

		if IsAuthenticated(context.Background()) {
			t.Error("expected IsAuthenticated to return false for empty context")
		}
	})

	t.Run("HasRole", func(t *testing.T) {
		roles := []string{"admin", "user"}
		ctx := context.WithValue(context.Background(), userRolesKey, roles)

		if !HasRole(ctx, "admin") {
			t.Error("expected HasRole to return true for 'admin'")
		}

		if HasRole(ctx, "superadmin") {
			t.Error("expected HasRole to return false for 'superadmin'")
		}
	})
}
