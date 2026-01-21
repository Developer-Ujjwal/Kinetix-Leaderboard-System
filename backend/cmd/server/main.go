package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/api/handlers"
	"backend/internal/config"
	"backend/internal/jobs"
	"backend/internal/repository"
	"backend/internal/service"
	"backend/internal/websocket"
	"backend/internal/worker"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize PostgreSQL with connection pooling
	db, err := initPostgres(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	log.Println("✓ Connected to PostgreSQL")

	// Initialize Redis
	redisClient, err := initRedis(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("✓ Connected to Redis")

	// Initialize repositories
	postgresRepo := repository.NewPostgresRepository(db)
	redisRepo := repository.NewRedisRepository(redisClient)

	// Run migrations
	if err := postgresRepo.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// Initialize Worker Pool for PostgreSQL persistence
	workerCount := 20     // Number of worker goroutines
	queueSize := 1000     // Buffered channel size
	workerPool := worker.NewWorkerPool(workerCount, queueSize, postgresRepo)
	workerPool.Start()

	// Initialize WebSocket Hub
	hub := websocket.NewHub(redisRepo, redisClient)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Initialize service with worker pool and redis client
	leaderboardService := service.NewLeaderboardService(redisRepo, postgresRepo, workerPool, redisClient)

	// Initialize Simulation Manager (high-performance internal job)
	simulatorConfig := jobs.SimulatorConfig{
		TickInterval:   500 * time.Millisecond, // 2 ticks/sec (slowed down)
		UpdatesPerTick: 1,                      // 1 update per tick = 2 updates/sec
		MinScoreChange: -50,
		MaxScoreChange: 50,
	}
	simulator := jobs.NewSimulationManager(leaderboardService, simulatorConfig)
	
	// Start simulator in background
	simCtx, simCancel := context.WithCancel(context.Background())
	defer simCancel()
	if err := simulator.Start(simCtx); err != nil {
		log.Printf("⚠️ Failed to start simulator: %v", err)
	}

	// Initialize handlers with hub
	leaderboardHandler := handlers.NewLeaderboardHandler(leaderboardService, hub)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Kinetix Leaderboard System",
		DisableStartupMessage: false,
		ErrorHandler:          customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// Routes
	api := app.Group("/api/v1")
	
	// Leaderboard routes
	api.Post("/scores", leaderboardHandler.UpdateScore)
	api.Get("/leaderboard", leaderboardHandler.GetLeaderboard)
	api.Get("/search/:username", leaderboardHandler.SearchUser)
	api.Get("/health", leaderboardHandler.HealthCheck)
	
	// Debug routes (load simulation)
	debug := api.Group("/debug")
	debug.Post("/simulate", leaderboardHandler.SimulateLoad)
	
	// WebSocket route with upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		// Check if it's a WebSocket upgrade request
		if fiberws.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws", fiberws.New(func(c *fiberws.Conn) {
		leaderboardHandler.HandleWebSocket(c)
	}))

	// Root route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Kinetix Leaderboard System API",
			"version": "1.0.0",
			"endpoints": []string{
				"POST /api/v1/scores",
				"GET /api/v1/leaderboard",
				"GET /api/v1/search/:username",
				"GET /api/v1/health",
				"POST /api/v1/debug/simulate",
				"WS /ws (WebSocket)",
			},
			"websocket_clients": hub.GetClientCount(),
		})
	})

	// Graceful shutdown with worker pool flushing
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("\n🛑 Shutting down server...")

		// First, stop simulator
		log.Println("⏹️ Stopping simulator...")
		simulator.Stop()

		// Second, stop accepting new HTTP requests
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.ShutdownWithContext(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}

		// Third, gracefully shutdown worker pool (flush pending writes)
		log.Println("🔄 Flushing worker pool (pending database writes)...")
		if err := workerPool.Shutdown(30 * time.Second); err != nil {
			log.Printf("Worker pool shutdown error: %v", err)
		}

		// Third, close database connections
		if err := postgresRepo.Close(); err != nil {
			log.Printf("Error closing PostgreSQL: %v", err)
		}
		if err := redisRepo.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}

		log.Println("✓ Server shutdown complete")
	}()

	// Start server
	port := cfg.Server.Port
	log.Printf("🚀 Server starting on port %d...", port)
	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// initPostgres initializes PostgreSQL connection with connection pooling
func initPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.GetDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, err
	}

	// Get underlying sql.DB for connection pooling
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Configure connection pool for worker pool (20 workers + buffer)
	// Max connections should be >= number of workers to prevent blocking
	sqlDB.SetMaxOpenConns(30)      // Allows 20 workers + 10 buffer for other operations
	sqlDB.SetMaxIdleConns(10)      // Keep some connections idle for reuse
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	log.Printf("✓ PostgreSQL connection pool configured: MaxOpen=%d, MaxIdle=%d", 30, 10)

	return db, nil
}

// initRedis initializes Redis connection with connection pooling
func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     20,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   "Request failed",
		"message": err.Error(),
	})
}
