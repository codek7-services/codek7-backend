package model

import "time"

type User struct {
	ID        string    `sql:"id"`
	Email     string    `sql:"email"`
	Username  string    `sql:"username"`
	Password  string    `sql:"password"`
	CreatedAt time.Time `sql:"created_at"`
}
