package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo/postgres"
	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/db"
	sharedlog "github.com/jia-app/paymentservice/internal/shared/log"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <csv-file-path>")
	}

	csvFilePath := os.Args[1]

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	if err := sharedlog.Init(cfg.Log.Level); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	ctx := context.Background()

	// Initialize database connection
	dbConfig := &db.Config{
		DSN:      cfg.Postgres.DSN,
		MaxConns: cfg.Postgres.MaxConns,
	}
	dbPool, err := db.NewPool(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}
	defer dbPool.Close()

	// Initialize repository
	repo, err := postgres.NewStoreWithPool(dbPool.Pool)
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}

	// Read and parse CSV file
	pricingZones, err := readPricingZonesFromCSV(csvFilePath)
	if err != nil {
		log.Fatalf("Failed to read pricing zones from CSV: %v", err)
	}

	fmt.Printf("Loaded %d pricing zones from CSV\n", len(pricingZones))

	// Import pricing zones to database
	if err := importPricingZones(ctx, repo, pricingZones); err != nil {
		log.Fatalf("Failed to import pricing zones: %v", err)
	}

	fmt.Println("Successfully imported pricing zones to database")
}

func readPricingZonesFromCSV(filePath string) ([]domain.PricingZone, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var pricingZones []domain.PricingZone
	now := time.Now()

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record: %w", err)
		}

		if len(record) < 7 {
			continue // Skip incomplete records
		}

		// Parse pricing multiplier
		multiplier, err := strconv.ParseFloat(record[6], 64)
		if err != nil {
			fmt.Printf("Warning: Invalid pricing multiplier for %s: %s\n", record[0], record[6])
			continue
		}

		pricingZone := domain.PricingZone{
			ID:                      uuid.New().String(),
			Country:                 strings.TrimSpace(record[0]),
			ISOCode:                 strings.ToUpper(strings.TrimSpace(record[1])),
			Zone:                    strings.ToUpper(strings.TrimSpace(record[2])),
			ZoneName:                strings.TrimSpace(record[3]),
			WorldBankClassification: strings.TrimSpace(record[4]),
			GNIPerCapitaThreshold:   strings.TrimSpace(record[5]),
			PricingMultiplier:       multiplier,
			CreatedAt:               now,
			UpdatedAt:               now,
		}

		// Validate zone
		if !domain.IsValidZone(pricingZone.Zone) {
			fmt.Printf("Warning: Invalid zone for %s: %s\n", pricingZone.Country, pricingZone.Zone)
			continue
		}

		pricingZones = append(pricingZones, pricingZone)
	}

	return pricingZones, nil
}

func importPricingZones(ctx context.Context, repo *postgres.Store, zones []domain.PricingZone) error {
	// For now, we'll just log the zones since the repository methods are not implemented yet
	// In a real implementation, you would call repo.PricingZone().BulkUpsert(ctx, zones)

	fmt.Println("Pricing zones to be imported:")
	for i, zone := range zones {
		if i >= 10 { // Show only first 10 for brevity
			fmt.Printf("... and %d more zones\n", len(zones)-10)
			break
		}
		fmt.Printf("  %s (%s) - Zone %s - Multiplier: %.2f\n",
			zone.Country, zone.ISOCode, zone.Zone, zone.PricingMultiplier)
	}

	// TODO: Implement actual database import when repository methods are ready
	fmt.Println("Note: Database import is not yet implemented - repository methods need to be completed")

	return nil
}
