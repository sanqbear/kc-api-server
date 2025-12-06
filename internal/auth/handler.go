package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/utils"
)

// Handler handles HTTP requests for authentication operations
type Handler struct {
	service Service
}

// NewHandler creates a new auth handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers auth routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	})
}

// RegisterProtectedRoutes registers auth routes that require authentication
func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/auth/me", h.Me)
	r.Post("/auth/logout-all", h.LogoutAll)
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account with email, password, and name. The user is automatically added to the 'public' group.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Registration data"
// @Success      201      {object}  RegisterResponse
// @Failure      400      {object}  ErrorResponse  "Invalid request or validation error"
// @Failure      409      {object}  ErrorResponse  "Email or login_id already exists"
// @Failure      500      {object}  ErrorResponse  "Internal server error"
// @Router       /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid request body")
		return
	}

	clientIP := getClientIP(r)
	userAgent := r.UserAgent()

	result, refreshToken, err := h.service.Register(r.Context(), &req, clientIP, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEmail):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid email format")
		case errors.Is(err, ErrInvalidName):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Name must have at least one locale value")
		case errors.Is(err, ErrInvalidPassword):
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Password must be at least 8 characters")
		case errors.Is(err, ErrEmailExists):
			utils.RespondError(w, r, http.StatusConflict, "Conflict", "Email already exists")
		case errors.Is(err, ErrLoginIDExists):
			utils.RespondError(w, r, http.StatusConflict, "Conflict", "Login ID already exists")
		case errors.Is(err, ErrPublicGroupNotFound):
			utils.RespondInternalError(w, r, err, "System configuration error")
		default:
			utils.RespondInternalError(w, r, err, "Failed to register user")
		}
		return
	}

	// Set refresh token as HTTP-only cookie
	setRefreshTokenCookie(w, refreshToken)

	utils.RespondJSON(w, http.StatusCreated, result)
}

// Login godoc
// @Summary      User login
// @Description  Authenticates a user with login_id/email and password. Returns access token in response body and refresh token as HTTP-only cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Login credentials"
// @Success      200      {object}  LoginResponse
// @Failure      400      {object}  ErrorResponse  "Invalid request"
// @Failure      401      {object}  ErrorResponse  "Invalid credentials"
// @Failure      500      {object}  ErrorResponse  "Internal server error"
// @Router       /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Invalid request body")
		return
	}

	clientIP := getClientIP(r)
	userAgent := r.UserAgent()

	result, refreshToken, err := h.service.Login(r.Context(), &req, clientIP, userAgent)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "Invalid credentials")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to login")
		return
	}

	// Set refresh token as HTTP-only cookie
	setRefreshTokenCookie(w, refreshToken)

	utils.RespondJSON(w, http.StatusOK, result)
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Uses the refresh token from HTTP-only cookie to generate new access and refresh tokens. Implements token rotation for security.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  TokenResponse
// @Failure      401  {object}  ErrorResponse  "Invalid or expired refresh token"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "Refresh token not found")
		return
	}

	clientIP := getClientIP(r)
	userAgent := r.UserAgent()

	result, newRefreshToken, err := h.service.Refresh(r.Context(), cookie.Value, clientIP, userAgent)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidToken):
			clearRefreshTokenCookie(w)
			utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "Invalid refresh token")
		case errors.Is(err, ErrTokenRevoked):
			clearRefreshTokenCookie(w)
			utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "Token has been revoked. Please login again.")
		case errors.Is(err, ErrTokenExpired):
			clearRefreshTokenCookie(w)
			utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "Refresh token has expired. Please login again.")
		default:
			utils.RespondInternalError(w, r, err, "Failed to refresh token")
		}
		return
	}

	// Set new refresh token as HTTP-only cookie
	setRefreshTokenCookie(w, newRefreshToken)

	utils.RespondJSON(w, http.StatusOK, result)
}

// Logout godoc
// @Summary      User logout
// @Description  Revokes the current refresh token and clears the cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  SuccessResponse
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Router       /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}

	// Clear the cookie regardless of logout result
	clearRefreshTokenCookie(w)

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Logged out successfully"})
}

// LogoutAll godoc
// @Summary      Logout from all devices
// @Description  Revokes all refresh tokens for the current user, logging them out from all devices.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  SuccessResponse
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /auth/logout-all [post]
func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "User not authenticated")
		return
	}

	if err := h.service.LogoutAll(r.Context(), userID); err != nil {
		utils.RespondInternalError(w, r, err, "Failed to logout from all devices")
		return
	}

	// Clear the cookie
	clearRefreshTokenCookie(w)

	utils.RespondJSON(w, http.StatusOK, SuccessResponse{Message: "Logged out from all devices successfully"})
}

// Me godoc
// @Summary      Get current user info
// @Description  Returns the currently authenticated user's information and roles.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  MeResponse
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      500  {object}  ErrorResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /auth/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		utils.RespondError(w, r, http.StatusUnauthorized, "Unauthorized", "User not authenticated")
		return
	}

	result, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to retrieve user information")
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

// setRefreshTokenCookie sets the refresh token as an HTTP-only cookie
func setRefreshTokenCookie(w http.ResponseWriter, token string) {
	secure := os.Getenv("APP_ENV") != "local"

	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     RefreshTokenCookiePath,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(RefreshTokenDuration.Seconds()),
		Expires:  time.Now().Add(RefreshTokenDuration),
	})
}

// clearRefreshTokenCookie clears the refresh token cookie
func clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     RefreshTokenCookiePath,
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") != "local",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return cleanIP(strings.TrimSpace(ips[0]))
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return cleanIP(xri)
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr

	// Handle IPv6 addresses with brackets (e.g., [::1]:port)
	if strings.HasPrefix(ip, "[") {
		if idx := strings.Index(ip, "]"); idx != -1 {
			ip = ip[1:idx] // Extract IP from brackets
		}
	} else {
		// Handle IPv4 addresses (e.g., 127.0.0.1:port)
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
	}

	return ip
}

// cleanIP removes brackets from IPv6 addresses if present
func cleanIP(ip string) string {
	ip = strings.TrimPrefix(ip, "[")
	ip = strings.TrimSuffix(ip, "]")
	// Also remove port if present after bracket removal
	if idx := strings.LastIndex(ip, ":"); idx != -1 && strings.Count(ip, ":") == 1 {
		// Only remove if it looks like a port (single colon for IPv4)
		ip = ip[:idx]
	}
	return ip
}

