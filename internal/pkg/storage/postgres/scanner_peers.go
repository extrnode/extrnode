package postgres

import (
	"fmt"
	"time"
)

type ScannerPeer struct {
	ID          int       `pg:"spr_id"`
	PeerID      int       `pg:"prs_id"` // Peer
	Date        time.Time `pg:"spr_date"`
	TimeConnect int       `pg:"spr_time_connect_ms"`
	IsAlive     bool      `pg:"spr_is_alive"`
}

const scannerPeersTable = "scanner.peers"

func (p *Storage) CreateScannerPeer(peerID int, date time.Time, timeConnect time.Duration, isAlive bool) error {
	if peerID == 0 {
		return fmt.Errorf("empty peerID")
	}

	query := `INSERT INTO scanner.peers (prs_id, spr_date, spr_time_connect_ms, spr_is_alive)
			VALUES (?, ?, ?, ?)`
	_, err := p.db.Exec(query, peerID, date.UTC(), timeConnect.Milliseconds(), isAlive)
	if err != nil {
		return err
	}

	return nil
}
