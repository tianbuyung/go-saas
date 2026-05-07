package domain

type User struct {
	ID            int64 `json:"-"`
	PublicID      string
	Name          string
	Email         string
	EmailVerified bool
	Image         string
}
