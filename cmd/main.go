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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"alx-wallet/internal/config"
	"alx-wallet/internal/handler"
	"alx-wallet/internal/middleware"
	"alx-wallet/internal/repository"
	"alx-wallet/internal/service"
	"alx-wallet/pkg/logger"
)

func main() {
	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// ── Logger ────────────────────────────────────────────────────────────────
	log, err := logger.New(cfg.Server.Mode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// ── Database ──────────────────────────────────────────────────────────────
	dbCfg, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		log.Fatal("invalid database DSN", zap.Error(err))
	}
	dbCfg.MaxConns = int32(cfg.Database.MaxOpenConns)
	dbCfg.MinConns = int32(cfg.Database.MaxIdleConns)
	dbCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	db, err := pgxpool.NewWithConfig(context.Background(), dbCfg)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("database ping failed", zap.Error(err))
	}
	log.Info("database connected")

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		// Redis is a cache — warn but don't crash the service.
		log.Warn("redis unavailable, cache disabled", zap.Error(err))
	} else {
		log.Info("redis connected")
	}
	defer rdb.Close()

	// ── Repositories ──────────────────────────────────────────────────────────
	accountRepo := repository.NewAccountRepo(db)
	ledgerRepo := repository.NewLedgerRepo(db)

	// ── Services ──────────────────────────────────────────────────────────────
	walletSvc := service.NewWalletService(accountRepo, ledgerRepo, rdb, log)

	// ── HTTP ──────────────────────────────────────────────────────────────────
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(
		middleware.Recovery(log),
		middleware.RequestID(),
		middleware.Logger(log),
	)

	// Health check (no auth required)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	walletHandler := handler.NewWalletHandler(walletSvc, log)
	walletHandler.RegisterRoutes(router)

	txRepo := repository.NewTransactionRepo(db)
	txHandler := handler.NewTransactionHandler(txRepo, log)
	txHandler.RegisterRoutes(router)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	go func() {
		log.Info("server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
	}
	log.Info("server stopped")
}
