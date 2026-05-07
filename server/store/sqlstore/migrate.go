package sqlstore

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/mattermost/morph"
	ps "github.com/mattermost/morph/drivers/postgres"
	mbindata "github.com/mattermost/morph/sources/embedded"
	"github.com/pkg/errors"
)

//go:embed migrations
var migrationsFS embed.FS

const (
	migrationsTable       = "rtk_db_migrations"
	legacyMigrationsTable = "rtk_schema_migrations"
)

// RunMigrations ensures the database schema is up to date.
// On first run after upgrading from the legacy hand-rolled system it drops the
// old rtk_schema_migrations tracking table; all migrations use IF NOT EXISTS so
// they are safe to re-apply against an existing schema.
func (s *Store) RunMigrations() error {
	if err := s.dropLegacyMigrationsTable(); err != nil {
		return fmt.Errorf("failed to drop legacy migrations table: %w", err)
	}

	engine, err := s.newMorphEngine()
	if err != nil {
		return fmt.Errorf("failed to create morph engine: %w", err)
	}
	defer engine.Close()

	if err := engine.ApplyAll(); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	if _, err := s.MaybeImportCallsChannels(); err != nil {
		return fmt.Errorf("failed to import calls_channels: %w", err)
	}

	return nil
}

func (s *Store) newMorphEngine() (*morph.Morph, error) {
	driver, err := ps.WithInstance(s.db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create postgres driver")
	}

	if err := driver.SetConfig("MigrationsTable", migrationsTable); err != nil {
		return nil, errors.Wrap(err, "failed to set migrations table name")
	}

	assetNames, err := sqlMigrationFileNames()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list migration files")
	}

	src, err := mbindata.WithInstance(&mbindata.AssetSource{
		Names: assetNames,
		AssetFunc: func(name string) ([]byte, error) {
			return migrationsFS.ReadFile(path.Join("migrations", "postgres", name))
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create migration source")
	}

	engine, err := morph.New(context.Background(), driver, src)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create morph engine")
	}

	return engine, nil
}

// sqlMigrationFileNames returns a sorted list of all migration filenames
// (both .up.sql and .down.sql) from the embedded postgres directory.
func sqlMigrationFileNames() ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, path.Join("migrations", "postgres"))
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// dropLegacyMigrationsTable removes the hand-rolled tracking table from
// earlier plugin versions. All current migrations are idempotent (IF NOT
// EXISTS), so morph can safely re-evaluate them against an existing schema.
func (s *Store) dropLegacyMigrationsTable() error {
	_, err := s.db.Exec(`DROP TABLE IF EXISTS ` + legacyMigrationsTable)
	return errors.Wrap(err, "failed to drop legacy migrations table")
}
