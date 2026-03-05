package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"alx-wallet/internal/handler"
	"alx-wallet/internal/repository"
	"alx-wallet/internal/service"
)

func main() {

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}

	accountRepo := repository.NewAccountRepository(db)
	ledgerRepo := repository.NewLedgerRepository(db)

	walletService := service.NewWalletService(accountRepo, ledgerRepo)
	walletHandler := handler.NewWalletHandler(walletService)

	r := gin.Default()

	r.POST("/accounts", walletHandler.CreateAccount)
	r.GET("/accounts/:id/balance", walletHandler.GetBalance)
	r.POST("/transfer", walletHandler.Transfer)

	log.Println("wallet service running on :8080")

	r.Run(":8080")
}
