package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	migrate "github.com/rubenv/sql-migrate"

	"extrnode-be/internal/pkg/config"
	"extrnode-be/internal/pkg/log"
)

const (
	driver = "sqlite3"
)

type Storage struct {
	ctx context.Context
	db  *sql.DB
}

func New(ctx context.Context, cfg config.SQLiteConfig) (s Storage, err error) {
	db, err := sql.Open(driver, fmt.Sprintf("file:%s?mode=rwc&_fk=1&_timeout=10000&_cache_size=-10000&_synchronous=NORMAL&_journal_mode=WAL", cfg.DBPath)) // cache=shared
	if err != nil {
		return s, fmt.Errorf("sql.Open: %s", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return s, fmt.Errorf("ping: %s", err)
	}

	migrations := &migrate.FileMigrationSource{
		Dir: cfg.MigrationsPath,
	}

	appliedMigrations, err := migrate.Exec(db, driver, migrations, migrate.Up)
	if err != nil {
		return s, fmt.Errorf("migrate.Exec: %s", err)
	}

	log.Logger.General.Infof("sqlite: applied migrations: %d", appliedMigrations)

	return Storage{
		ctx: ctx,
		db:  db,
	}, nil
}
