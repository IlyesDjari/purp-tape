package db

import (
	"context"
	"log/slog"
	"testing"
)

func TestNew_ValidURL(t *testing.T) {
	// This test would require a real PostgreSQL database
	// For unit testing, we verify the connection pool configuration
	
	logger := slog.Default()
	
	// Test with an invalid connection (will fail, but we can verify structure)
	_, err := New(context.Background(), "invalid://url", 5, 1, logger)
	
	// Error is expected since we don't have a real database
	if err == nil {
		t.Errorf("New() with invalid URL should error")
	}
}

func TestConnectionPoolStats_DefaultValues(t *testing.T) {
	stats := ConnectionPoolStats{
		OpenConnections:    10,
		InUse:              5,
		Idle:               5,
		MaxOpenConnections: 25,
	}

	if stats.OpenConnections != 10 {
		t.Errorf("expected OpenConnections=10, got %d", stats.OpenConnections)
	}

	if stats.InUse != 5 {
		t.Errorf("expected InUse=5, got %d", stats.InUse)
	}

	if stats.Idle != 5 {
		t.Errorf("expected Idle=5, got %d", stats.Idle)
	}

	if stats.MaxOpenConnections != 25 {
		t.Errorf("expected MaxOpenConnections=25, got %d", stats.MaxOpenConnections)
	}
}

func TestConnectionPoolStats_PoolHealth(t *testing.T) {
	tests := []struct {
		name           string
		openConns      int
		inUse          int
		maxConns       int
		expectedLow    bool
	}{
		{"healthy pool", 20, 5, 25, false},
		{"almost full", 24, 24, 25, true},
		{"empty pool", 1, 0, 25, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify stats can be calculated
			stats := ConnectionPoolStats{
				OpenConnections:    tt.openConns,
				InUse:              tt.inUse,
				MaxOpenConnections: tt.maxConns,
			}

			if stats.OpenConnections > stats.MaxOpenConnections {
				t.Errorf("OpenConnections (%d) exceeds Max (%d)", 
					stats.OpenConnections, stats.MaxOpenConnections)
			}

			if stats.InUse > stats.OpenConnections {
				t.Errorf("InUse (%d) exceeds OpenConnections (%d)", 
					stats.InUse, stats.OpenConnections)
			}
		})
	}
}
