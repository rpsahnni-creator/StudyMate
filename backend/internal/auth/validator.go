package auth

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
)

var (
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong     = errors.New("password must be at most 72 characters")
	ErrPasswordComplexity  = errors.New("password must contain uppercase, lowercase, and a number")
	ErrPasswordTooCommon   = errors.New("password is too common")
	ErrEmailInvalid        = errors.New("invalid email address")
	ErrEmailTooLong        = errors.New("email must be at most 254 characters")
	ErrMobileInvalid       = errors.New("invalid mobile number; use 10-digit Indian mobile")
	ErrClassRequired       = errors.New("class is required")
)

// ValidatePassword enforces password strength rules.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 72 {
		return ErrPasswordTooLong
	}

	hasUpper, hasLower, hasDigit := false, false, false
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return ErrPasswordComplexity
	}
	if isCommonPassword(strings.ToLower(password)) {
		return ErrPasswordTooCommon
	}
	return nil
}

// ValidateEmail checks RFC 5322 format and normalizes to lowercase trimmed form.
func ValidateEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", ErrEmailInvalid
	}
	if len(email) > 254 {
		return "", ErrEmailTooLong
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrEmailInvalid
	}
	return email, nil
}

// ValidateMobile normalizes and validates a 10-digit Indian mobile number.
func ValidateMobile(mobile string) (string, error) {
	mobile = strings.TrimSpace(mobile)
	mobile = strings.ReplaceAll(mobile, " ", "")
	mobile = strings.ReplaceAll(mobile, "-", "")
	if strings.HasPrefix(mobile, "+91") {
		mobile = mobile[3:]
	}
	if strings.HasPrefix(mobile, "91") && len(mobile) == 12 {
		mobile = mobile[2:]
	}
	if len(mobile) != 10 {
		return "", ErrMobileInvalid
	}
	for _, r := range mobile {
		if r < '0' || r > '9' {
			return "", ErrMobileInvalid
		}
	}
	if mobile[0] < '6' {
		return "", ErrMobileInvalid
	}
	return mobile, nil
}

// ValidateClass ensures a non-empty class label for students.
func ValidateClass(class string) (string, error) {
	class = SanitizeUserInput(class)
	if class == "" {
		return "", ErrClassRequired
	}
	if len(class) > 50 {
		class = class[:50]
	}
	return class, nil
}

// SanitizeUserInput trims, strips null bytes, and caps length.
func SanitizeUserInput(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.TrimSpace(s)
	if len(s) > 1000 {
		s = s[:1000]
	}
	return s
}

func isCommonPassword(password string) bool {
	_, ok := commonPasswords[password]
	return ok
}

// Top 100 common passwords (lowercase).
var commonPasswords = map[string]struct{}{
	"password": {}, "password1": {}, "password12": {}, "password123": {}, "password1234": {},
	"123456": {}, "1234567": {}, "12345678": {}, "123456789": {}, "1234567890": {},
	"qwerty": {}, "qwerty123": {}, "qwertyuiop": {}, "abc123": {}, "111111": {},
	"123123": {}, "12345": {}, "1234": {}, "000000": {}, "654321": {},
	"666666": {}, "696969": {}, "7777777": {}, "888888": {}, "999999": {},
	"iloveyou": {}, "admin": {}, "admin123": {}, "letmein": {}, "welcome": {},
	"welcome1": {}, "monkey": {}, "dragon": {}, "master": {}, "login": {},
	"princess": {}, "football": {}, "shadow": {}, "sunshine": {}, "superman": {},
	"michael": {}, "jennifer": {}, "trustno1": {}, "batman": {}, "access": {},
	"hello": {}, "charlie": {}, "donald": {}, "mustang": {}, "baseball": {},
	"starwars": {}, "whatever": {}, "freedom": {}, "passw0rd": {}, "qazwsx": {},
	"zaq12wsx": {}, "asdfgh": {}, "ashley": {}, "bailey": {}, "pass": {},
	"test": {}, "test123": {}, "guest": {}, "root": {}, "toor": {},
	"changeme": {}, "secret": {}, "default": {}, "computer": {}, "internet": {},
	"cookie": {}, "pepper": {}, "ginger": {}, "jordan": {}, "hunter": {},
	"ranger": {}, "buster": {}, "thomas": {}, "robert": {}, "daniel": {},
	"matthew": {}, "andrew": {}, "joshua": {}, "summer": {}, "winter": {},
	"spring": {}, "autumn": {}, "flower": {}, "cheese": {}, "coffee": {},
	"orange": {}, "apple": {}, "banana": {}, "purple": {}, "yellow": {},
	"silver": {}, "golden": {}, "soccer": {}, "hockey": {}, "tennis": {},
	"guitar": {}, "music": {}, "killer": {}, "matrix": {}, "ninja": {},
}
