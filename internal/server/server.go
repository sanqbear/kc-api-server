package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"kc-api/internal/aiqueue"
	"kc-api/internal/auth"
	"kc-api/internal/commoncodes"
	"kc-api/internal/database"
	"kc-api/internal/departments"
	"kc-api/internal/files"
	"kc-api/internal/groups"
	"kc-api/internal/plugins/ews"
	"kc-api/internal/rbac"
	"kc-api/internal/roles"
	"kc-api/internal/tickets"
	"kc-api/internal/users"
)

type Server struct {
	port int

	db                database.Service
	userHandler       *users.Handler
	authHandler       *auth.Handler
	authMiddleware    *auth.Middleware
	rbacHandler       *rbac.Handler
	rbacMiddleware    *rbac.Middleware
	permissionManager *rbac.PermissionManager
	ticketHandler     *tickets.Handler
	fileHandler       *files.Handler
	ewsHandler        *ews.Handler
	aiQueueHandler    *aiqueue.Handler

	// Organization management handlers
	commonCodeHandler *commoncodes.Handler
	roleHandler       *roles.Handler
	departmentHandler *departments.Handler
	groupHandler      *groups.Handler
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

	// Initialize RBAC domain with DI
	rbacRepo := rbac.NewRepository(db.DB())
	permissionManager := rbac.NewPermissionManager(rbacRepo)
	rbacHandler := rbac.NewHandler(permissionManager)
	rbacMiddleware := rbac.NewMiddleware(permissionManager)

	// Load initial permissions from database
	if err := permissionManager.LoadPermissions(context.Background()); err != nil {
		log.Printf("Warning: Failed to load initial permissions: %v", err)
	}

	// Initialize tickets domain with DI
	ticketRepo := tickets.NewRepository(db.DB())
	ticketService := tickets.NewService(ticketRepo)
	ticketHandler := tickets.NewHandler(ticketService)

	// Initialize files domain with DI
	fileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	if fileStoragePath == "" {
		fileStoragePath = "./uploads"
	}
	fileRepo := files.NewRepository(db.DB())
	fileService := files.NewService(fileRepo, fileStoragePath)
	fileHandler := files.NewHandler(fileService)

	// Initialize EWS plugin (optional)
	var ewsHandler *ews.Handler
	ewsConfig, err := ews.LoadConfig()
	if err != nil {
		log.Printf("Warning: Failed to load EWS config: %v", err)
	} else if ewsConfig != nil {
		ewsClient, err := ews.NewClient(ewsConfig)
		if err != nil {
			log.Printf("Warning: Failed to create EWS client: %v", err)
		} else {
			ewsHandler = ews.NewHandler(ewsClient)
			log.Println("EWS plugin initialized successfully")
		}
	} else {
		log.Println("EWS plugin not configured (EWS_SERVER_URL not set)")
	}

	// Initialize AI queue (optional)
	var aiQueueHandler *aiqueue.Handler
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr != "" {
		redisPassword := os.Getenv("REDIS_PASSWORD")
		redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
		redisQueueName := os.Getenv("REDIS_QUEUE_NAME")
		if redisQueueName == "" {
			redisQueueName = "celery"
		}

		aiQueueClient, err := aiqueue.NewClient(redisAddr, redisPassword, redisDB, redisQueueName)
		if err != nil {
			log.Printf("Warning: Failed to create AI queue client: %v", err)
		} else {
			aiQueueService := aiqueue.NewService(aiQueueClient)
			aiQueueHandler = aiqueue.NewHandler(aiQueueService)
			log.Println("AI queue integration initialized successfully")
		}
	} else {
		log.Println("AI queue integration not configured (REDIS_ADDR not set)")
	}

	// Initialize common codes domain with DI
	commonCodeRepo := commoncodes.NewRepository(db.DB())
	commonCodeService := commoncodes.NewService(commonCodeRepo)
	commonCodeHandler := commoncodes.NewHandler(commonCodeService)

	// Initialize roles domain with DI
	roleRepo := roles.NewRepository(db.DB())
	roleService := roles.NewService(roleRepo)
	roleHandler := roles.NewHandler(roleService)

	// Initialize departments domain with DI
	departmentRepo := departments.NewRepository(db.DB())
	departmentService := departments.NewService(departmentRepo)
	departmentHandler := departments.NewHandler(departmentService)

	// Initialize groups domain with DI
	groupRepo := groups.NewRepository(db.DB())
	groupService := groups.NewService(groupRepo)
	groupHandler := groups.NewHandler(groupService)

	NewServer := &Server{
		port:              port,
		db:                db,
		userHandler:       userHandler,
		authHandler:       authHandler,
		authMiddleware:    authMiddleware,
		rbacHandler:       rbacHandler,
		rbacMiddleware:    rbacMiddleware,
		permissionManager: permissionManager,
		ticketHandler:     ticketHandler,
		fileHandler:       fileHandler,
		ewsHandler:        ewsHandler,
		aiQueueHandler:    aiQueueHandler,

		// Organization management handlers
		commonCodeHandler: commonCodeHandler,
		roleHandler:       roleHandler,
		departmentHandler: departmentHandler,
		groupHandler:      groupHandler,
	}

	log.Printf("Server starting on port %d", NewServer.port)
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
