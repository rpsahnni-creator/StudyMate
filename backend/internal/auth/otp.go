package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

const otpDigits = 6

// GenerateNumericOTP returns a 6-digit OTP string.
func GenerateNumericOTP() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(b[:]) % 1000000
	return fmt.Sprintf("%0*d", otpDigits, n), nil
}

// HashOTP stores a one-way hash of the OTP for database lookup.
func HashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
