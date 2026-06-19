package migrations

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runner handles database migrations using golang-migrate
type Runner struct {
	migrationsPath string
	pool           *pgxpool.Pool
}

// NewRunner creates a new migration runner
// If migrationsPath is relative, it will be converted to an absolute path based on the current working directory
func NewRunner(pool *pgxpool.Pool, migrationsPath string) *Runner {
	// Convert relative paths to absolute
	if !filepath.IsAbs(migrationsPath) {
		wd, err := os.Getwd()
		if err == nil {
			migrationsPath = filepath.Join(wd, migrationsPath)
		}
	}

	return &Runner{
		migrationsPath: migrationsPath,
		pool:           pool,
	}
}

// buildFileURL creates a proper file:// URL for golang-migrate on both Windows and Unix
// golang-migrate requires:
// - Windows: file://C:/path (path has drive letter)
// - Unix: file:///path (path starts with /)
func buildFileURL(path string) string {
	// Convert backslashes to forward slashes
	path = filepath.ToSlash(path)
	// Simply prepend file:// - works for both platforms
	return "file://" + path
}

// Up runs all pending migrations
func (r *Runner) Up() error {
	// Get connection config to build DSN
	connConfig := r.pool.Config().ConnConfig

	// Build DSN from config
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		connConfig.User,
		connConfig.Password,
		connConfig.Host,
		connConfig.Port,
		connConfig.Database,
		func() string {
			if connConfig.TLSConfig != nil {
				return "require"
			}
			return "disable"
		}(),
	)

	// Create migration instance from file source
	m, err := migrate.New(
		buildFileURL(r.migrationsPath),
		dsn,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Run all pending migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("Migrations completed successfully")
	return nil
}

// Down rolls back all migrations
func (r *Runner) Down() error {
	// Get connection config to build DSN
	connConfig := r.pool.Config().ConnConfig

	// Build DSN from config
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		connConfig.User,
		connConfig.Password,
		connConfig.Host,
		connConfig.Port,
		connConfig.Database,
		func() string {
			if connConfig.TLSConfig != nil {
				return "require"
			}
			return "disable"
		}(),
	)

	// Create migration instance from file source
	m, err := migrate.New(
		buildFileURL(r.migrationsPath),
		dsn,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Rollback all migrations
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	fmt.Println("Rollback completed successfully")
	return nil
}

// Steps runs a specific number of migrations
func (r *Runner) Steps(steps int) error {
	if steps == 0 {
		return fmt.Errorf("steps must be non-zero")
	}

	// Get connection config to build DSN
	connConfig := r.pool.Config().ConnConfig

	// Build DSN from config
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		connConfig.User,
		connConfig.Password,
		connConfig.Host,
		connConfig.Port,
		connConfig.Database,
		func() string {
			if connConfig.TLSConfig != nil {
				return "require"
			}
			return "disable"
		}(),
	)

	// Create migration instance from file source
	m, err := migrate.New(
		buildFileURL(r.migrationsPath),
		dsn,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Run specific number of migrations
	if err := m.Steps(steps); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Printf("Applied %d migration steps\n", steps)
	return nil
}
