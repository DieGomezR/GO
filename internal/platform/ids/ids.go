// Package ids concentra la generacion de identificadores simples para las
// entidades del proyecto.
package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// New crea un ID legible con prefijo, timestamp y aleatorio corto.
func New(prefix string) string {
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixMilli(), randomHex(4))
}

// OrderNumber crea un numero comercial mas amigable para mostrar en ordenes.
func OrderNumber() string {
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102-150405"), randomHex(2))
}

// randomHex genera un sufijo aleatorio en hexadecimal.
func randomHex(size int) string {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "fallback"
	}

	return hex.EncodeToString(buffer)
}
