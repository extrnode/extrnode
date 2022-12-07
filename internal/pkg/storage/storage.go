package storage

import (
	"context"
	"fmt"
	"time"

	"extrnode-be/internal/pkg/config"

	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	log "github.com/sirupsen/logrus"
)

type PgStorage struct {
	db   orm.DB
	isTx bool
}

func New(ctx context.Context, cfg config.PostgresConfig) (s PgStorage, err error) {
	db := pg.Connect(&pg.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		User:            cfg.User,
		Password:        cfg.Pass,
		Database:        cfg.Database,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		PoolTimeout:     20 * time.Second,
		ApplicationName: "extrnode-go",
	})

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

	return PgStorage{
		db: db,
	}, nil
}

func (p *PgStorage) BeginTx() (s PgStorage, err error) {
	if p.isTx {
		return s, fmt.Errorf("already tx")
	}

	tx, err := p.db.(*pg.DB).Begin()
	if err != nil {
		return s, err
	}

	return PgStorage{
		db:   tx,
		isTx: true,
	}, nil
}

func (p *PgStorage) Rollback() error {
	if !p.isTx {
		return fmt.Errorf("not tx")
	}

	return p.db.(*pg.Tx).Rollback()
}

func (p *PgStorage) Commit() error {
	if !p.isTx {
		return fmt.Errorf("not tx")
	}

	return p.db.(*pg.Tx).Commit()
}
