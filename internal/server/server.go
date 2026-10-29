package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/DoWithLogic/golang-clean-architecture/config"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/app_gin"
	"github.com/DoWithLogic/golang-clean-architecture/pkg/datasources"
	appRedis "github.com/DoWithLogic/golang-clean-architecture/pkg/redis"
	"github.com/labstack/gommon/log"

	"github.com/samber/lo"

	"github.com/go-redis/redis/v8"
)

type Server struct {
	db          *gorm.DB        // Database connection.
	gin         *app_gin.Engine // Gin HTTP server instance.
	cfg         config.Config   // Configuration settings for the application.
	redisClient *redis.Client
}

func NewServer(ctx context.Context, cfg config.Config) *Server {
	serverOpts := []app_gin.GinOptionFn{}
	if cfg.Observability.Enable {
		serverOpts = append(serverOpts, app_gin.WithTracing(cfg.App.Name))
	}

	return &Server{
		db:          lo.Must(datasources.NewMySQLDB(ctx, cfg.Database)),
		gin:         cfg.Server.New(serverOpts...),
		cfg:         cfg,
		redisClient: appRedis.NewRedisClient(ctx, cfg.Redis),
	}
}

func (s *Server) Run() error {
	if err := s.setup(); err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", s.cfg.Server.Port),
		Handler: s.gin,
	}

	// Set up signal handling to gracefully shutdown the server upon receiving a SIGTERM or SIGINT signal.
	// Using a buffered channel with capacity 1 to ensure signals are not missed.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM)
	signal.Notify(quit, syscall.SIGINT)

	// Start a goroutine to handle the shutdown signal.
	go func() {
		// Wait for the signal from the quit channel.
		<-quit

		// Log the shutdown process.
		log.Info("Server is shutting down...")

		// Create a context with a timeout of 10 seconds for the server shutdown.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Close DB Connection
		lo.Must(s.db.DB()).Close()

		// Shutdown gracefully.
		httpServer.Shutdown(ctx)
	}()

	// Start the gin server and listen on the configured port.
	return httpServer.ListenAndServe()
}
