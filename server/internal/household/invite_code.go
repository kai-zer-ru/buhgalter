package household

import (
	"crypto/rand"
	"strings"
	"unicode"
)

const inviteAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const inviteRawLen = 9

func generateInviteCode() (string, error) {
	out := make([]byte, inviteRawLen)
	n := len(inviteAlphabet)
	limit := 256 - (256 % n)
	buf := make([]byte, 1)
	for i := 0; i < inviteRawLen; i++ {
		for {
			if _, err := rand.Read(buf); err != nil {
				return "", err
			}
			if int(buf[0]) >= limit {
				continue
			}
			out[i] = inviteAlphabet[int(buf[0])%n]
			break
		}
	}
	return string(out), nil
}

func formatInviteCode(raw string) string {
	if len(raw) != inviteRawLen {
		return raw
	}
	return raw[:3] + "-" + raw[3:6] + "-" + raw[6:]
}

func normalizeInviteCode(input string) (string, bool) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(input) {
		if r == '-' || unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	raw := b.String()
	if len(raw) != inviteRawLen {
		return "", false
	}
	for _, c := range raw {
		if !strings.ContainsRune(inviteAlphabet, c) {
			return "", false
		}
	}
	return raw, true
}
