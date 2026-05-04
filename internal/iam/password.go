package iam

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

type PasswordHash struct {
	Hash string
	Salt string
}

func GeneratePassword(password string) (*PasswordHash, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	return &PasswordHash{
		Hash: base64.StdEncoding.EncodeToString(hash),
		Salt: base64.StdEncoding.EncodeToString(salt),
	}, nil
}

func ComparePassword(password, hash, salt string) bool {
	saltBytes, _ := base64.StdEncoding.DecodeString(salt)
	hashBytes, _ := base64.StdEncoding.DecodeString(hash)

	newHash := argon2.IDKey([]byte(password), saltBytes, 1, 64*1024, 4, 32)

	return string(newHash) == string(hashBytes)
}
