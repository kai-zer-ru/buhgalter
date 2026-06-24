package user

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

const apiTokenPrefix = "bhg_"

func generateAPIToken() (raw, hash, prefix string, err error) {
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}
	raw = apiTokenPrefix + base64.RawURLEncoding.EncodeToString(b)
	if len(raw) < 8 {
		return "", "", "", fmt.Errorf("token too short")
	}
	prefix = raw[:8]
	hash = auth.HashToken(raw)
	return raw, hash, prefix, nil
}
