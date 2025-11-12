package database

import (
	"context"
	"fmt"
	"time"

	"github.com/Rizwank123/notification_system/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	pool *pgxpool.Pool
	log  logger.Logger
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

func NewPostgresDB(ctx context.Context, cfg Config, log logger.Logger) (*PostgresDB, error) {
	// Build connection string
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Set pool configuration
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection pool established",
		"max_conns", cfg.MaxConns,
		"min_conns", cfg.MinConns,
	)

	return &PostgresDB{
		pool: pool,
		log:  log,
	}, nil
}

func (p *PostgresDB) Close() {
	p.log.Info("Closing database connection pool")
	p.pool.Close()
}

func (p *PostgresDB) Pool() *pgxpool.Pool {
	return p.pool
}

// Health check
func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Stats returns pool statistics
func (p *PostgresDB) Stats() map[string]interface{} {
	stat := p.pool.Stat()
	return map[string]interface{}{
		"total_conns":        stat.TotalConns(),
		"idle_conns":         stat.IdleConns(),
		"acquired_conns":     stat.AcquiredConns(),
		"constructing_conns": stat.ConstructingConns(),
	}
}
