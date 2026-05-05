package domain

type Account struct {
	ID         int64
	UserID     int64
	AccountID  string
	ProviderID string
	Password   string
	Salt       string
}
