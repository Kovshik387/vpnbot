package model

import "time"

type Cooldown struct {
	UserID    int64     `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
}
