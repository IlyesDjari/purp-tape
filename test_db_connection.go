package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := "postgresql://postgres:cowHyp-vifzo8-hifrux@db.bwibsdocyylxzyhjvrjz.supabase.co:5432/postgres"
	
	fmt.Println("Testing database connection...")
	fmt.Printf("Connecting to: db.bwibsdocyylxzyhjvrjz.supabase.co:5432\n")

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Printf("❌ Failed to parse config: %v\n", err)
		return
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		fmt.Printf("❌ Failed to create pool: %v\n", err)
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("❌ Failed to ping database: %v\n", err)
		return
	}

	var version string
	err = pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		fmt.Printf("❌ Failed to query: %v\n", err)
		return
	}

	fmt.Printf("✅ Connection successful!\n")
	fmt.Printf("PostgreSQL version: %s\n", version)
}
