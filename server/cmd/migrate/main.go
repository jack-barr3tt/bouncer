package main

import (
	"context"
	"log"
	"os"

	"github.com/jack-barr3tt/bouncer/db"
	"github.com/jack-barr3tt/bouncer/internal/envfile"
)

func main() {
	if err := envfile.Load(); err != nil {
		log.Fatal(err)
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}
}
