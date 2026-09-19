package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/apps/ofertas-scraper/internal/application"
	"github.com/Arthur-Bamberg/orquestrador-ofertas-supermercados/modules/ofertas-store"
)

func main() {
	log.SetFlags(0)
	if err := godotenv.Load(); err != nil {
		log.Printf("could not load .env: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL env var is not set")
	}

	ctx := context.Background()
	catalog, err := store.Open(ctx, dbURL)
	if err != nil {
		log.Fatalf("could not open catalog: %v", err)
	}
	defer catalog.Close()

	service := application.NewMergeService(catalog)

	err = service.MergeProducts(ctx)
	if err != nil {
		log.Fatalf("merge failed: %v", err)
	}
	fmt.Println("Product merge completed successfully.")
}
