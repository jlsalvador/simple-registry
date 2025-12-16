package rbac

import "golang.org/x/crypto/bcrypt"

type User struct {
	Name         string
	PasswordHash string
	Groups       []string
}

func (u *User) IsPasswordValid(pwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pwd)) == nil
}
