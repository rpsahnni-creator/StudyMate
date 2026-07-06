// One-off admin promotion. Usage: go run ./scripts/promote_admin.go kijitechnology@gmail.com
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"studyapp/backend/internal/bootstrap"
	"studyapp/backend/internal/common/config"
)

func main() {
	email := "kijitechnology@gmail.com"
	if len(os.Args) > 1 {
		email = os.Args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	promoted, err := bootstrap.PromoteAdminByEmail(ctx, pool, email)
	if err != nil {
		log.Fatal(err)
	}
	if promoted {
		fmt.Printf("promoted %s to admin — re-login for JWT role update\n", email)
	} else {
		fmt.Printf("%s is already admin or user not found\n", email)
	}
}
