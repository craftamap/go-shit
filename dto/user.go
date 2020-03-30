package dto

import (
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID         int `gorm:"PRIMARY_KEY,UNIQUE"`
	Username   string
	PasswdHash string
}

func (m *User) SetPasswd(passwd string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(passwd), 14)
	m.PasswdHash = string(bytes)
	return err
}

func (m *User) CheckPasswd(passwd string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(m.PasswdHash), []byte(passwd))
	return err == nil
}
