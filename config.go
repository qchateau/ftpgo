package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

// EncryptPassword encrypts a clear password into a salted hash
func EncryptPassword(pass []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
}

// LoadConfig loads the configuration file
func LoadConfig(path string) (Config, error) {
	var c Config
	if content, err := ioutil.ReadFile(path); err != nil {
		return c, err
	} else if err := json.Unmarshal(content, &c); err != nil {
		return c, err
	}
	if !filepath.IsAbs(c.Jail) {
		return c, errors.New("fail path must be absolute")
	}
	c.Jail = filepath.Clean(c.Jail)
	return c, nil
}

// Config contains the server configuration
type Config struct {
	Login, Password, Addr, Jail string
	Port                        int
}

// AllowAnyUser returns true if users can connect using any username
func (c *Config) AllowAnyUser() bool {
	return c.Login == ""
}

// PasswordRequired returns true if a password is required for authentication
func (c *Config) PasswordRequired() bool {
	return !c.VerifyPassword("")
}

// VerifyPassword returns true if the password given matches the one in the config
func (c *Config) VerifyPassword(pass string) bool {
	return bcrypt.CompareHashAndPassword(
		[]byte(c.Password), []byte(pass)) == nil
}

// Dump writes configuration to a file
func (c *Config) Dump(path string) (err error) {
	content, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, content, 0644)
	return
}
