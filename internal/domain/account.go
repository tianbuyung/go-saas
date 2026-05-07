package domain

type Account struct {
	ID         int64  `json:"-"`
	UserID     int64  `json:"-"`
	AccountID  string
	ProviderID string
	Password   string `json:"-"`
	Salt       string `json:"-"`
}
