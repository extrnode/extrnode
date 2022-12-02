package storage

import (
	"crypto/tls"
	"extrnode-be/internal/pkg/config"
	"fmt"

	"github.com/go-pg/migrations"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const rowsAffected = 0

type Storage interface {
	BeginTx() (TxStorage, error)

	// CreateAsset(asset types.Asset) (types.Asset, error)
}

type TxStorage interface {
	Rollback() error
	Commit() error

	Storage
}

type pgStorage struct {
	conn *pg.DB
	db   orm.DB
}

type pgTxStorage struct {
	pgStorage
}

func New(cfg config.PostgresConfig) (Storage, error) {
	db := pg.Connect(&pg.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		User:     cfg.User,
		Password: cfg.Pass,
		Database: cfg.Database,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	var oldVersion, newVersion int64
	var err error

	collection := migrations.NewCollection()
	collection.DisableSQLAutodiscover(true)
	err = collection.DiscoverSQLMigrations(cfg.MigrationsPath)
	if err != nil {
		return nil, errors.Wrap(err, "migrations error")
	}

	collection.Run(db, "init")
	oldVersion, newVersion, err = collection.Run(db, "up")
	if err != nil {
		return nil, err
	}

	if newVersion != oldVersion {
		log.Infof("PG migrated from version %d to %d", oldVersion, newVersion)
	} else {
		log.Infof("PG migration version is %d", oldVersion)
	}

	var d pgStorage
	d.conn = db
	d.db = db

	return &d, nil
}

func (p *pgStorage) BeginTx() (TxStorage, error) {
	tx, err := p.conn.Begin()
	if err != nil {
		return nil, err
	}

	return &pgTxStorage{
		pgStorage: pgStorage{
			db: tx,
		},
	}, nil
}

func (p *pgTxStorage) Rollback() error {
	return p.db.(*pg.Tx).Rollback()
}

func (p *pgTxStorage) Commit() error {
	return p.db.(*pg.Tx).Commit()
}
