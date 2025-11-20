package repositories

import (
	"context"
	"fmt"
	utils "hephaestus/internal/utils"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (r *Repository) CreateConnection(cfg *utils.Config) (*pgxpool.Pool, error) {
	r.log.Debug("Create connection................")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)

	r.log.Debug("Connect dsn: ", dsn)
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = 10 // max connection count
	poolConfig.HealthCheckPeriod = 30 * time.Second
	r.log.Debug("Max connections: ", poolConfig.MaxConns)
	r.log.Debug("Health check: ", poolConfig.HealthCheckPeriod, " seconds")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		r.log.Error("Error creating connection pool: ", err)
		return nil, err
	}
	r.log.Debug("Connection pool created.")

	if dbPool == nil {
		r.log.Error("Database pool is nil, check connection parameters")
		return nil, fmt.Errorf("failed to create database connection")
	}

	r.log.Debug("Ping database...")
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, err
	}

	r.log.Debug("connected.")

	if cfg.Logger.LogLevel == "debug" {
		r.checkConnection(dbPool)
	}
	return dbPool, nil
}

func (r *Repository) checkConnection(dbPool *pgxpool.Pool) {
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		r.log.Error("Error creating transaction: ", err)
		return
	}
	defer tx.Rollback(context.Background())

	var dbName string
	err = tx.QueryRow(context.Background(), "SELECT current_database()").Scan(&dbName)
	if err != nil {
		r.log.Error("Error getting current database: ", err)
	} else {
		r.log.Debug("Connected to database: ", dbName)
	}

	var searchPath string
	err = tx.QueryRow(context.Background(), "SHOW search_path").Scan(&searchPath)
	if err != nil {
		r.log.Error("Error getting search_path: ", err)
	} else {
		r.log.Debug("Current search_path: ", searchPath)
	}
}
