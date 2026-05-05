package domain

type UserAuth struct {
	ID       int64
	Email    string
	Password string
	Salt     string
}
