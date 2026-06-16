package hover

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// totp generates a 6-digit TOTP code from a base32 secret.
func totp(secret string) (string, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	if p := len(secret) % 8; p != 0 {
		secret += strings.Repeat("=", 8-p)
	}
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("invalid TOTP secret: %w", err)
	}

	counter := uint64(time.Now().Unix() / 30)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	h := mac.Sum(nil)

	offset := h[len(h)-1] & 0xf
	code := binary.BigEndian.Uint32(h[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", code%1_000_000), nil
}
