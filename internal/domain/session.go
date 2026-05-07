package domain

import "time"

type Session struct {
	ID        int64     `json:"-"`
	UserID    int64     `json:"-"`
	Token     string
	ExpiresAt time.Time
	IPAddress string
	UserAgent string
}
