package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", s.HelloWorldHandler)

	r.Get("/health", s.healthHandler)

	// Register public auth routes (login, register, refresh, logout)
	s.authHandler.RegisterRoutes(r)

	// Protected routes requiring authentication and RBAC authorization
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware.Authenticate)
		r.Use(s.rbacMiddleware.Authorize)

		// Protected auth routes (me, logout-all)
		s.authHandler.RegisterProtectedRoutes(r)

		// Protected user routes
		s.userHandler.RegisterRoutes(r)

		// RBAC admin routes (refresh-permissions) - requires sysadmin role via RBAC
		s.rbacHandler.RegisterRoutes(r)

		// Protected ticket routes
		s.ticketHandler.RegisterRoutes(r)
	})

	// Swagger UI route
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	return r
}

// HelloWorldHandler godoc
// @Summary      Hello World
// @Description  Returns a hello world message
// @Tags         general
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       / [get]
func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

// healthHandler godoc
// @Summary      Health Check
// @Description  Returns the health status of the service and database connection
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, _ := json.Marshal(s.db.Health())
	_, _ = w.Write(jsonResp)
}
