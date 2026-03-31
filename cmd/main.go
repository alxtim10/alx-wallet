package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"os/signal"
	"syscall"

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

	// ── Context (for shutdown signals) ─────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := initDB(cfg, log)
	if err != nil {
		log.Fatal("database initialization failed", zap.Error(err))
	}
	defer db.Close()
	log.Info("database connected")

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Warn("redis unavailable, cache disabled", zap.Error(err))
	} else {
		log.Info("redis connected")
	}
	defer rdb.Close()

	// ── Repositories ──────────────────────────────────────────────────────────
	accountRepo := repository.NewAccountRepo(db)
	ledgerRepo := repository.NewLedgerRepo(db)
	txRepo := repository.NewTransactionRepo(db)

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

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	handler.NewWalletHandler(walletSvc, log).RegisterRoutes(router)
	handler.NewTransactionHandler(txRepo, log).RegisterRoutes(router)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// ── Start server ──────────────────────────────────────────────────────────
	go func() {
		log.Info("server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", zap.Error(err))
		}
	}()

	// ── Wait for shutdown signal ──────────────────────────────────────────────
	<-ctx.Done()
	log.Info("shutdown signal received")

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("shutting down server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced shutdown", zap.Error(err))
	}

	log.Info("server stopped")
}

// ── DB Initialization with Retry ────────────────────────────────────────────
func initDB(cfg *config.Config, log *zap.Logger) (*pgxpool.Pool, error) {
	dbCfg, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}

	dbCfg.MaxConns = int32(cfg.Database.MaxOpenConns)
	dbCfg.MinConns = int32(cfg.Database.MaxIdleConns)
	dbCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	db, err := pgxpool.NewWithConfig(context.Background(), dbCfg)
	if err != nil {
		return nil, err
	}

	// Retry logic
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < 10; i++ {
		if err := db.Ping(ctx); err == nil {
			return db, nil
		} else {
			log.Warn("database not ready, retrying...",
				zap.Int("attempt", i+1),
				zap.Error(err),
			)
		}
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("database not reachable after retries")
}
