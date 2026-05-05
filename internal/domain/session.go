package domain

import "time"

type Session struct {
	ID           int64
	UserID       int64
	Token        string
	RefreshToken string
	ExpiresAt    time.Time
	IPAddress    string
	UserAgent    string
}
