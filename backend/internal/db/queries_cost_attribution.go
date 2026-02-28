package db

import (
	"context"
	"database/sql"
	"time"
)

// ============================================================================
// COST ATTRIBUTION MODELS
// ============================================================================

// CostAttribution represents storage costs for a user/project/period
type CostAttribution struct {
	ID              string     `db:"id"`
	UserID          string     `db:"user_id"`
	ProjectID       string     `db:"project_id"`
	TimePeriod      time.Time  `db:"time_period"`
	StorageGBHours  float64    `db:"storage_gb_hours"`
	StorageCostUSD  float64    `db:"storage_cost_usd"`
	APICallCount    int        `db:"api_call_count"`
	APICostUSD      float64    `db:"api_cost_usd"`
	TransferGB      int        `db:"transfer_gb"`
	TransferCostUSD float64    `db:"transfer_cost_usd"`
	TotalCostUSD    float64    `db:"total_cost_usd"`
}

// MonthlyUserCosts represents aggregated costs for a user in a month
type MonthlyUserCosts struct {
	UserID               string    `db:"user_id"`
	BillingMonth         time.Time `db:"billing_month"`
	TotalStorageCostUSD  float64   `db:"total_storage_cost_usd"`
	TotalAPICostUSD      float64   `db:"total_api_cost_usd"`
	TotalTransferCostUSD float64   `db:"total_transfer_cost_usd"`
	TotalCostUSD         float64   `db:"total_cost_usd"`
	ProjectsCount        int       `db:"projects_count"`
	LastActivityDate     time.Time `db:"last_activity_date"`
}

// ProjectCostDetail represents costs for a project on a specific day
type ProjectCostDetail struct {
	TimePeriod      time.Time `db:"time_period"`
	StorageCostUSD  float64   `db:"storage_cost_usd"`
	APICostUSD      float64   `db:"api_cost_usd"`
	TransferCostUSD float64   `db:"transfer_cost_usd"`
	TotalCostUSD    float64   `db:"total_cost_usd"`
	StorageGBHours  float64   `db:"storage_gb_hours"`
}

// CostBreakdown shows project-level cost distribution for a user
type CostBreakdown struct {
	ProjectID       string  `db:"project_id"`
	ProjectName     string  `db:"project_name"`
	StorageCostUSD  float64 `db:"storage_cost_usd"`
	APICostUSD      float64 `db:"api_cost_usd"`
	TransferCostUSD float64 `db:"transfer_cost_usd"`
	TotalCostUSD    float64 `db:"total_cost_usd"`
	CostPercentage  float64 `db:"cost_percentage"`
}

// UserInvoice represents billable charges for a user in a month
type UserInvoice struct {
	InvoiceMonth       string  `db:"invoice_month"`
	TotalAmountUSD     float64 `db:"total_amount_usd"`
	StorageAmountUSD   float64 `db:"storage_amount_usd"`
	APIAmountUSD       float64 `db:"api_amount_usd"`
	TransferAmountUSD  float64 `db:"transfer_amount_usd"`
	BreakdownJSON      string  `db:"breakdown"`
}

// ============================================================================
// COST ATTRIBUTION QUERIES
// ============================================================================

// GetUserCurrentMonthCost retrieves user's cost so far this month (O(1) with function)
func (db *Database) GetUserCurrentMonthCost(ctx context.Context, userID string) (float64, error) {
	var cost float64
	err := db.pool.QueryRow(ctx,
		`SELECT user_current_month_cost($1)`,
		userID,
	).Scan(&cost)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return cost, err
}

// GetProjectCurrentMonthCost retrieves project's cost so far this month
func (db *Database) GetProjectCurrentMonthCost(ctx context.Context, projectID string) (float64, error) {
	var cost float64
	err := db.pool.QueryRow(ctx,
		`SELECT project_current_month_cost($1)`,
		projectID,
	).Scan(&cost)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return cost, err
}

// GetUserMonthlyCosts retrieves user's costs for the last N months (for trend analysis)
func (db *Database) GetUserMonthlyCosts(ctx context.Context, userID string, months int) ([]MonthlyUserCosts, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT user_id, billing_month, total_storage_cost_usd, total_api_cost_usd, 
		        total_transfer_cost_usd, total_cost_usd, projects_count, last_activity_date
		 FROM mv_user_monthly_costs
		 WHERE user_id = $1
		 ORDER BY billing_month DESC
		 LIMIT $2`,
		userID, months,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var costs []MonthlyUserCosts
	for rows.Next() {
		var c MonthlyUserCosts
		if err := rows.Scan(&c.UserID, &c.BillingMonth, &c.TotalStorageCostUSD, &c.TotalAPICostUSD,
			&c.TotalTransferCostUSD, &c.TotalCostUSD, &c.ProjectsCount, &c.LastActivityDate); err != nil {
			return nil, err
		}
		costs = append(costs, c)
	}
	return costs, rows.Err()
}

// GetProjectMonthlyCosts retrieves project's costs for the last N months
func (db *Database) GetProjectMonthlyCosts(ctx context.Context, projectID string, months int) ([]ProjectCostDetail, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT time_period, storage_cost_usd, api_cost_usd, transfer_cost_usd, total_cost_usd, storage_gb_hours
		 FROM project_cost_details($1, CURRENT_DATE)
		 LIMIT $2`,
		projectID, months*30,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var costs []ProjectCostDetail
	for rows.Next() {
		var c ProjectCostDetail
		if err := rows.Scan(&c.TimePeriod, &c.StorageCostUSD, &c.APICostUSD,
			&c.TransferCostUSD, &c.TotalCostUSD, &c.StorageGBHours); err != nil {
			return nil, err
		}
		costs = append(costs, c)
	}
	return costs, rows.Err()
}

// GetUserCostBreakdown returns per-project cost distribution for a user in a month
func (db *Database) GetUserCostBreakdown(ctx context.Context, userID string, month time.Time) ([]CostBreakdown, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT project_id, project_name, storage_cost_usd, api_cost_usd, 
		        transfer_cost_usd, total_cost_usd, cost_percentage
		 FROM user_cost_breakdown($1, $2)`,
		userID, month,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdown []CostBreakdown
	for rows.Next() {
		var b CostBreakdown
		if err := rows.Scan(&b.ProjectID, &b.ProjectName, &b.StorageCostUSD, &b.APICostUSD,
			&b.TransferCostUSD, &b.TotalCostUSD, &b.CostPercentage); err != nil {
			return nil, err
		}
		breakdown = append(breakdown, b)
	}
	return breakdown, rows.Err()
}

// GenerateUserInvoice creates an invoice for a user for a specific month
func (db *Database) GenerateUserInvoice(ctx context.Context, userID string, month time.Time) (*UserInvoice, error) {
	var invoice UserInvoice
	err := db.pool.QueryRow(ctx,
		`SELECT invoice_month, total_amount_usd, storage_amount_usd, api_amount_usd, transfer_amount_usd, breakdown
		 FROM generate_user_invoice($1, $2)`,
		userID, month,
	).Scan(&invoice.InvoiceMonth, &invoice.TotalAmountUSD, &invoice.StorageAmountUSD,
		&invoice.APIAmountUSD, &invoice.TransferAmountUSD, &invoice.BreakdownJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

// GetAllUsersCurrentMonthCosts retrieves all users with their current month costs (for admin dashboard)
func (db *Database) GetAllUsersCurrentMonthCosts(ctx context.Context) (map[string]float64, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT user_id, total_cost_usd FROM v_current_month_user_costs`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	costs := make(map[string]float64)
	for rows.Next() {
		var userID string
		var cost float64
		if err := rows.Scan(&userID, &cost); err != nil {
			return nil, err
		}
		costs[userID] = cost
	}
	return costs, rows.Err()
}

// IdentifyHighCostProjects returns top N projects by cost for a user
func (db *Database) IdentifyHighCostProjects(ctx context.Context, userID string, limit int) ([]CostBreakdown, error) {
	breakdown, err := db.GetUserCostBreakdown(ctx, userID, time.Now())
	if err != nil {
		return nil, err
	}

	if len(breakdown) > limit {
		breakdown = breakdown[:limit]
	}
	return breakdown, nil
}

// RecordCostAttribution stores daily cost data for a project/user pair
func (db *Database) RecordCostAttribution(ctx context.Context, ca *CostAttribution) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO cost_attribution 
			(user_id, project_id, time_period, storage_gb_hours, storage_cost_usd, 
			 api_call_count, api_cost_usd, transfer_gb, transfer_cost_usd)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (user_id, project_id, time_period) 
		 DO UPDATE SET
			storage_gb_hours = $4,
			storage_cost_usd = $5,
			api_call_count = $6,
			api_cost_usd = $7,
			transfer_gb = $8,
			transfer_cost_usd = $9`,
		ca.UserID, ca.ProjectID, ca.TimePeriod, ca.StorageGBHours, ca.StorageCostUSD,
		ca.APICallCount, ca.APICostUSD, ca.TransferGB, ca.TransferCostUSD,
	)
	return err
}
