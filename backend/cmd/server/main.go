package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"paygate-omni/config"
	"paygate-omni/internal/controller"
	"paygate-omni/internal/middleware"
	"paygate-omni/internal/model"
	"paygate-omni/internal/repository"
	"paygate-omni/internal/service"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	db, err := connectDB(cfg, logger)
	if err != nil {
		logger.Fatal("failed to connect to PostgreSQL", zap.Error(err))
	}

	if err := db.AutoMigrate(
		&model.Merchant{},
		&model.PayChannel{},
		&model.Order{},
	); err != nil {
		logger.Fatal("database auto-migration failed", zap.Error(err))
	}

	rdb, err := connectRedis(cfg, logger)
	if err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}

	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.StructuredLogger(logger))

	// DI Inject
	store := &repository.Store{DB: db, RDB: rdb}
	paySvc := service.NewPayService(store, logger)
	
	registerRoutes(engine, cfg, db, rdb, paySvc, logger)

	addr := ":" + cfg.Server.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("PayGate-Omni server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server listen error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutdown signal received", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
	logger.Info("PayGate-Omni server stopped cleanly")
}

func registerRoutes(engine *gin.Engine, cfg *config.Config, db *gorm.DB, rdb *redis.Client, paySvc *service.PayService, logger *zap.Logger) {
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Unix()})
	})

	secretKeyFn := middleware.SecretKeyFn(func(appID string) (string, error) {
		var m model.Merchant
		if err := db.Where("app_id = ? AND is_active = true", appID).First(&m).Error; err != nil {
			return "", fmt.Errorf("merchant not found: %w", err)
		}
		return m.SecretKey, nil
	})

	payCtrl := controller.NewPayController(logger, paySvc)
	epayCtrl := controller.NewEpayController(logger, db, paySvc)
	adminCtrl := controller.NewAdminController(logger, cfg, db, rdb)

	engine.Any("/submit.php", epayCtrl.Submit)
	engine.Any("/api.php", epayCtrl.API)

	v1 := engine.Group("/api/v1")
	{
		merchantPay := v1.Group("/pay")
		merchantPay.Use(middleware.SignatureVerifier(secretKeyFn, rdb, logger))
		merchantPay.POST("/create", payCtrl.CreatePay)

		v1.POST("/pay/notify/:channel", payCtrl.NotifyPay)

		admin := v1.Group("/admin")
		admin.POST("/login", adminCtrl.Login)
		
		adminProtected := admin.Group("")
		adminProtected.Use(middleware.AdminAuth(rdb))
		adminProtected.GET("/stats", adminCtrl.GetStats)
adminProtected.GET("/merchants", adminCtrl.ListMerchants)
adminProtected.POST("/merchants", adminCtrl.CreateMerchant)
adminProtected.PUT("/merchants/:id", adminCtrl.UpdateMerchant)
adminProtected.DELETE("/merchants/:id", adminCtrl.DeleteMerchant)
		adminProtected.GET("/channels", adminCtrl.ListChannels)
adminProtected.POST("/channels", adminCtrl.CreateChannel)
adminProtected.PUT("/channels/:id", adminCtrl.UpdateChannel)
		adminProtected.DELETE("/channels/:id", adminCtrl.DeleteChannel)
                adminProtected.GET("/orders", adminCtrl.ListOrders)
	}
        
        // Serve frontend static files (must be after API routes to avoid conflicts)
        // Serve frontend static files via NoRoute handler
        engine.NoRoute(func(c *gin.Context) {
                // Try to serve file from frontend/dist directory
                filePath := "./frontend/dist" + c.Request.URL.Path
                if _, err := os.Stat(filePath); err == nil {
                        // File exists, serve it
                        c.File(filePath)
                } else {
                        // File not found, serve index.html for SPA routing
                        c.File("./frontend/dist/index.html")
                }
        })
}

func connectDB(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	gormLogLevel := gormlogger.Info
	if cfg.Server.Env == "production" {
		gormLogLevel = gormlogger.Error
	}
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), &gorm.Config{Logger: gormlogger.Default.LogMode(gormLogLevel)})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func connectRedis(cfg *config.Config, logger *zap.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return rdb, nil
}
