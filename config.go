package main

import (
	"encoding/json"
	"io/ioutil"

	"golang.org/x/crypto/bcrypt"
)

func EncryptPassword(pass []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
}

func LoadConfig(path string) (c Config, err error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, &c)
	return
}

// Config contains the server configuration
type Config struct {
	Login, Password, Addr string
	Port                  int
}

func (c *Config) AllowAnyUser() bool {
	return c.Login == ""
}

func (c *Config) PasswordRequired() bool {
	return !c.VerifyPassword("")
}

func (c *Config) VerifyPassword(pass string) bool {
	return bcrypt.CompareHashAndPassword(
		[]byte(c.Password), []byte(pass)) == nil
}

func (c *Config) Dump(path string) (err error) {
	content, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, content, 0644)
	return
}
