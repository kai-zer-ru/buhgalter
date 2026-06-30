package user

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

const apiTokenPrefix = "bhg_"

// defaultTokenLifetime is the expiry when the client does not set never_expires or expires_at.
const defaultTokenLifetime = 30 * 24 * time.Hour

var errTokenExpiresInvalid = errors.New("invalid expires_at")

func resolveTokenExpiry(neverExpires bool, expiresAt *string, now time.Time) (dbValue any, responsePtr *string, perpetual bool, err error) {
	if neverExpires {
		return nil, nil, true, nil
	}
	if expiresAt != nil && *expiresAt != "" {
		t, parseErr := time.Parse(time.RFC3339, *expiresAt)
		if parseErr != nil {
			return nil, nil, false, errTokenExpiresInvalid
		}
		if !t.After(now.UTC()) {
			return nil, nil, false, errTokenExpiresInvalid
		}
		s := t.UTC().Format(time.RFC3339)
		return s, &s, false, nil
	}
	t := now.UTC().Add(defaultTokenLifetime)
	s := t.Format(time.RFC3339)
	return s, &s, false, nil
}

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
