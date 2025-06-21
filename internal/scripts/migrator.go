package scripts

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/issafronov/shortener/internal/middleware/logger"
	"go.uber.org/zap"
)

// RunMigrations запускает миграции
func RunMigrations(databaseURI string) error {
	m, err := migrate.New(
		"file://internal/scripts/migrations",
		databaseURI,
	)
	if err != nil {
		logger.Log.Error("failed to initialize migrate", zap.Error(err))
		return fmt.Errorf("failed to init migrate: %w", err)
	}

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Log.Info("no migrations to apply")
			return nil
		}
		logger.Log.Error("failed to apply migrations", zap.Error(err))
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	logger.Log.Info("migrations applied successfully")
	return nil
}
