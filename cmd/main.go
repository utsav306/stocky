package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stockBackend/internal/controllers"
	"stockBackend/internal/db"
	"stockBackend/internal/repository"
	"stockBackend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Global variables - keeping these at package level for easy access across the app
var (
	log          *logrus.Logger           // Logger instance for the entire application
	dbPool       *pgxpool.Pool            // Database connection pool
	priceService *services.PriceService   // Price service for stock price updates
)

func init() {
	// Setup logger first thing - we'll need this for debugging
	log = logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	// Try to load .env file - it's okay if it doesn't exist in production
	if err := godotenv.Load(); err != nil {
		log.Warn("No .env file found, using system environment variables")
	}

	// Allow log level to be configured via environment
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		if parsedLevel, err := logrus.ParseLevel(level); err == nil {
			log.SetLevel(parsedLevel)
		}
	}

	// Support both JSON and text formats for logs
	if format := os.Getenv("LOG_FORMAT"); format == "text" {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
}

func main() {
	log.Info("Starting Stock Reward Backend Service...")

	// Initialize database connection
	if err := initDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbPool.Close()

	// Initialize database wrapper
	db.InitDB(dbPool)

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbPool)
	stockPriceRepo := repository.NewStockPriceRepository(dbPool)
	rewardRepo := repository.NewRewardRepository(dbPool)
	ledgerRepo := repository.NewLedgerRepository(dbPool)
	rewardRequestRepo := repository.NewRewardRequestRepository(dbPool)
	portfolioRepo := repository.NewPortfolioRepository(dbPool)

	// Initialize services
	priceService = services.NewPriceService(stockPriceRepo, log)
	rewardService := services.NewRewardService(
		rewardRepo,
		ledgerRepo,
		rewardRequestRepo,
		userRepo,
		priceService,
		log,
	)
	portfolioService := services.NewPortfolioService(portfolioRepo, rewardRepo, log)

	// Start price service
	if err := priceService.Start(); err != nil {
		log.Fatalf("Failed to start price service: %v", err)
	}
	defer priceService.Stop()

	// Initialize controllers
	priceController := controllers.NewPriceController(priceService, log)
	rewardController := controllers.NewRewardController(rewardService, log)
	portfolioController := controllers.NewPortfolioController(portfolioService, log)

	// Set Gin mode
	if mode := os.Getenv("GIN_MODE"); mode != "" {
		gin.SetMode(mode)
	}

	// Create Gin router
	router := gin.New()

	// Middleware
	router.Use(ginLogger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Register routes
	registerRoutes(router, priceController, rewardController, portfolioController)

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited")
}

// initDatabase sets up our PostgreSQL connection
// Supports both Supabase connection strings and traditional individual params
func initDatabase() error {
	var connString string
	
	// First, check if we have a full DATABASE_URL (common for Supabase, Heroku, etc.)
	databaseURL := os.Getenv("DATABASE_URL")
	
	if databaseURL != "" {
		// Great! We have a connection string, just use it as-is
		connString = databaseURL
		log.Info("Using DATABASE_URL for database connection")
	} else {
		// No DATABASE_URL found, let's build the connection string from individual params
		// Fall back to individual environment variables
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		dbSSLMode := os.Getenv("DB_SSLMODE")

		// Set sensible defaults for local development
		if dbHost == "" {
			dbHost = "localhost"
		}
		if dbPort == "" {
			dbPort = "5432" // Standard PostgreSQL port
		}
		if dbName == "" {
			dbName = "assignment"
		}
		if dbSSLMode == "" {
			dbSSLMode = "disable" // Disable SSL for local dev
		}

		// Build the connection string manually
		connString = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode,
		)
		log.Info("Using individual DB_* environment variables for database connection")
	}

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("unable to parse database config: %w", err)
	}

	// Configure connection pool settings
	// These numbers work well for most applications, adjust based on your needs
	config.MaxConns = 10                      // Max 10 concurrent connections
	config.MinConns = 2                       // Keep 2 connections warm
	config.MaxConnLifetime = time.Hour        // Recycle connections after 1 hour
	config.MaxConnIdleTime = 30 * time.Minute // Close idle connections after 30 min

	// Give ourselves 10 seconds to connect
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Quick ping to make sure we can actually talk to the database
	if err := dbPool.Ping(ctx); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	log.Info("Database connection established successfully")
	return nil
}

// registerRoutes sets up all our API endpoints
func registerRoutes(
	router *gin.Engine,
	priceController *controllers.PriceController,
	rewardController *controllers.RewardController,
	portfolioController *controllers.PortfolioController,
) {
	// Basic health check endpoint - useful for monitoring
	router.GET("/health", healthCheckHandler)

	// All our main API routes under /api/v1
	v1 := router.Group("/api/v1")
	{
		// Stock price related endpoints
		v1.POST("/prices/update", priceController.TriggerPriceUpdate)
		v1.POST("/prices/update/:symbol", priceController.UpdateSingleStockPrice)
		v1.GET("/prices/:symbol", priceController.GetLatestPrice)
		v1.GET("/prices/:symbol/history", priceController.GetPriceHistory)
		v1.GET("/prices/stocks", priceController.GetSupportedStocks)

		// Reward management endpoints
		v1.POST("/reward", rewardController.CreateReward)
		v1.GET("/reward/:eventId", rewardController.GetRewardByEventID)
		v1.GET("/rewards/:userId", rewardController.GetUserRewards)

		// Portfolio and analytics endpoints
		v1.GET("/today-stocks/:userId", portfolioController.GetTodayStocks)
		v1.GET("/historical-inr/:userId", portfolioController.GetHistoricalINR)
		v1.GET("/stats/:userId", portfolioController.GetUserStats)
		v1.GET("/portfolio/:userId", portfolioController.GetUserPortfolio)
		v1.GET("/holdings/:userId", portfolioController.GetDailyHoldings)
	}

	log.Info("Routes registered successfully")
}

// healthCheckHandler returns the health status of the service
// Useful for load balancers and monitoring tools
func healthCheckHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to ping the database to make sure it's alive
	dbStatus := "healthy"
	if err := dbPool.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
		log.Errorf("Database health check failed: %v", err)
	}

	// Return 503 if database is down, otherwise 200
	status := http.StatusOK
	if dbStatus == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"database":  dbStatus,
		"service":   "stock-reward-backend",
		"version":   "1.0.0",
	})
}

// ginLogger is our custom logging middleware
// Logs every request with useful info like latency, status code, etc.
func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record when the request started
		startTime := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Let the request proceed
		c.Next()

		// Calculate how long it took
		latency := time.Since(startTime)

		// Get status code
		statusCode := c.Writer.Status()

		// Build log entry
		entry := log.WithFields(logrus.Fields{
			"status":     statusCode,
			"method":     c.Request.Method,
			"path":       path,
			"query":      raw,
			"ip":         c.ClientIP(),
			"latency":    latency.Milliseconds(),
			"user_agent": c.Request.UserAgent(),
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.String())
		} else {
			entry.Info("Request processed")
		}
	}
}

// corsMiddleware handles CORS headers so our API can be called from browsers
// TODO: In production, replace "*" with specific allowed origins for security
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow requests from any origin (fine for development)
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
