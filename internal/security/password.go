// Package security encapsula operaciones minimas relacionadas con credenciales.
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// NewSalt genera un salt aleatorio por usuario.
// Este proyecto usa un hasher educativo para evitar dependencias externas.
// Antes de produccion debe cambiarse por Argon2id o bcrypt.
func NewSalt() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}

// HashPassword combina salt y password en un hash SHA-256 simple.
func HashPassword(password, salt string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

// ComparePassword compara el hash esperado con el calculado en tiempo constante.
func ComparePassword(password, salt, expectedHash string) bool {
	currentHash := HashPassword(password, salt)
	return subtle.ConstantTimeCompare([]byte(currentHash), []byte(expectedHash)) == 1
}
