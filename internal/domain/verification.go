package domain

import "time"

type Verification struct {
	ID         int64
	Identifier string
	Value      string
	Type       string
	ExpiresAt  time.Time
	UpdatedAt  time.Time
}
