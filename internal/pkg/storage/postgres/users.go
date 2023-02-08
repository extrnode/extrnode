package postgres

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
)

type User struct {
	ID         int64     `pg:"usr_id"`
	ProviderID string    `pg:"usr_provider_id"`
	ApiToken   uuid.UUID `pg:"usr_api_token"`
}

const userTable = "users"

func (p *Storage) GetOrCreateUser(providerId string) (u User, err error) {
	if providerId == "" {
		return u, fmt.Errorf("empty providerId")
	}

	query, args, err := sq.Select("usr_id, usr_provider_id, usr_api_token").
		From(userTable).
		Where("usr_provider_id = ?", providerId).ToSql()
	if err != nil {
		return u, err
	}

	s, err := p.BeginTx()
	if err != nil {
		return u, fmt.Errorf("beginTx: %s", err)
	}
	defer s.Rollback()

	_, err = s.db.QueryOne(&u, query, args...)
	if err != nil && err != pg.ErrNoRows {
		return u, fmt.Errorf("select: %s", err)
	}

	if err == pg.ErrNoRows {
		apiToken, err := uuid.NewRandom()
		if err != nil {
			return u, fmt.Errorf("uuid.New: %s", err)
		}

		query = `INSERT INTO users (usr_provider_id, usr_api_token)
			VALUES (?, ?) RETURNING usr_id, usr_provider_id, usr_api_token`

		_, err = p.db.QueryOne(&u, query, providerId, apiToken)
		if err != nil {
			return u, fmt.Errorf("insert: %s", err)
		}
	}

	err = s.Commit()
	if err != nil {
		return u, fmt.Errorf("commit: %s", err)
	}

	return u, nil
}
