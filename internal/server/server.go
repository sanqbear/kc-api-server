package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"kc-api/internal/auth"
	"kc-api/internal/database"
	"kc-api/internal/users"
)

type Server struct {
	port int

	db             database.Service
	userHandler    *users.Handler
	authHandler    *auth.Handler
	authMiddleware *auth.Middleware
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = "default-encryption-key-change-in-production"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-jwt-secret-change-in-production"
	}

	// Initialize database
	db := database.New()

	// Initialize user domain with DI
	userRepo := users.NewRepository(db.DB())
	userService := users.NewService(userRepo, encryptionKey)
	userHandler := users.NewHandler(userService)

	// Initialize auth domain with DI
	authRepo := auth.NewRepository(db.DB())
	authService := auth.NewService(authRepo, jwtSecret)
	authHandler := auth.NewHandler(authService)
	authMiddleware := auth.NewMiddleware(authService)

	NewServer := &Server{
		port:           port,
		db:             db,
		userHandler:    userHandler,
		authHandler:    authHandler,
		authMiddleware: authMiddleware,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
