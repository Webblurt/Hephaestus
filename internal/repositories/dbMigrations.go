package repositories

import (
	"fmt"
	utils "hephaestus/internal/utils"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func (r *Repository) RunMigrations(cfg *utils.Config) error {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	migrationsPath := "file://" + filepath.ToSlash(filepath.Join(cfg.Database.MigrationPath, "migrations"))

	r.log.Debug("Migrations path:", migrationsPath)

	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return err
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return err
	}

	r.log.Debug("Current migration version:", version)

	if dirty {
		r.log.Warn("Database is in a dirty migration state! Manual intervention may be required.")
		return fmt.Errorf("database is in a dirty state, please check the migration history")
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			r.log.Debug("No new migrations to apply")
			return nil
		}
		return fmt.Errorf("migration error: %w", err)
	}

	r.log.Info("Migrations applied successfully")
	return nil
}
