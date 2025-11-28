package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"kc-api/internal/database"
	"kc-api/internal/users"
)

type Server struct {
	port int

	db          database.Service
	userHandler *users.Handler
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = "default-encryption-key-change-in-production"
	}

	// Initialize database
	db := database.New()

	// Initialize user domain with DI
	userRepo := users.NewRepository(db.DB())
	userService := users.NewService(userRepo, encryptionKey)
	userHandler := users.NewHandler(userService)

	NewServer := &Server{
		port:        port,
		db:          db,
		userHandler: userHandler,
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
