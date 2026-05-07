package domain

import "time"

type Verification struct {
	ID         int64 `json:"-"`
	Identifier string
	Value      string
	Type       string
	ExpiresAt  time.Time
	UpdatedAt  time.Time
}
