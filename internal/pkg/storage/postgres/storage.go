package postgres

import (
	"context"
	"fmt"
	"time"

	"extrnode-be/internal/pkg/config_types"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	log "github.com/sirupsen/logrus"
)

type Storage struct {
	db   orm.DB
	isTx bool
}

func New(ctx context.Context, cfg config_types.PostgresConfig) (s Storage, err error) {
	// DialTimeout default is 5s
	db := pg.Connect(&pg.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		User:     cfg.User,
		Password: cfg.Pass,
		Database: cfg.DB,
	}).
		WithContext(ctx).
		WithTimeout(5 * time.Second)

	err = db.Ping(ctx)
	if err != nil {
		return s, fmt.Errorf("ping: %s", err)
	}

	collection := migrations.NewCollection()
	collection.DisableSQLAutodiscover(true)
	err = collection.DiscoverSQLMigrations(cfg.MigrationsPath)
	if err != nil {
		return s, fmt.Errorf("DiscoverSQLMigrations: %s", err)
	}

	err = db.RunInTransaction(ctx, func(tx *pg.Tx) (err error) {
		_, _, err = collection.Run(db, "init")
		if err != nil {
			return err
		}
		return
	})
	if err != nil {
		return s, fmt.Errorf("init migration: %s", err)
	}
	var oldVersion, newVersion int64
	err = db.RunInTransaction(ctx, func(tx *pg.Tx) (err error) {
		oldVersion, newVersion, err = collection.Run(db, "up")
		if err != nil {
			return err
		}
		return
	})
	if err != nil {
		return s, fmt.Errorf("migration: %s", err)
	}

	if newVersion != oldVersion {
		log.Infof("PG migrated from version %d to %d", oldVersion, newVersion)
	} else {
		log.Infof("PG migration version is %d", oldVersion)
	}

	return Storage{
		db: db,
	}, nil
}

func (p *Storage) BeginTx() (s Storage, err error) {
	if p.isTx {
		return s, fmt.Errorf("already tx")
	}

	tx, err := p.db.(*pg.DB).Begin()
	if err != nil {
		return s, err
	}

	return Storage{
		db:   tx,
		isTx: true,
	}, nil
}

func (p *Storage) Rollback() error {
	if !p.isTx {
		return fmt.Errorf("not tx")
	}

	return p.db.(*pg.Tx).Rollback()
}

func (p *Storage) Commit() error {
	if !p.isTx {
		return fmt.Errorf("not tx")
	}

	return p.db.(*pg.Tx).Commit()
}
