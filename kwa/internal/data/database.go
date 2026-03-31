package data

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"mthesis/kwa/internal/config"
	"mthesis/kwa/internal/entity"
)

// Service defines the data operations required by export orchestration.
type Service interface {
	// GetPhaseMetricsByID returns parsed-ready phase metric rows for a specific run.
	GetPhaseMetricsByID(ctx context.Context, runID string) ([]entity.PhaseMetrics, error)
	// GetPhaseMetricsBatch returns paginated phase metric rows across all runs.
	GetPhaseMetricsBatch(ctx context.Context, limit, offset int) ([]entity.PhaseMetrics, error)
	// GetMetricKeys returns the full ordered metric-key set used for CSV headers.
	GetMetricKeys(ctx context.Context) ([]string, error)
	Close() error
}

type service struct {
	db       *sql.DB
	database string
}

// New opens and validates a PostgreSQL connection for kwa data access.
func New(cfg config.DatabaseConfig) (Service, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Schema,
	)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("open database connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Printf("database connection established")

	return &service{db: db, database: cfg.Database}, nil
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("closing database connection")
	return s.db.Close()
}
