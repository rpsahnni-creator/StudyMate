package auth

import "testing"

func TestValidatePassword_CommonPasswordRejected(t *testing.T) {
	err := ValidatePassword("password123")
	if err != ErrPasswordTooCommon && err != ErrPasswordComplexity {
		t.Fatalf("expected common or complexity error, got %v", err)
	}
}

func TestValidatePassword_ValidPassword(t *testing.T) {
	if err := ValidatePassword("StrongPass123"); err != nil {
		t.Fatalf("expected valid password, got %v", err)
	}
}

func TestValidatePassword_TooShort(t *testing.T) {
	if err := ValidatePassword("Ab1"); err != ErrPasswordTooShort {
		t.Fatalf("expected too short, got %v", err)
	}
}

func TestValidateEmail_Normalizes(t *testing.T) {
	email, err := ValidateEmail("  User@Example.COM ")
	if err != nil {
		t.Fatal(err)
	}
	if email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", email)
	}
}

func TestSanitizeUserInput(t *testing.T) {
	got := SanitizeUserInput("  hello\x00world  ")
	if got != "helloworld" {
		t.Fatalf("unexpected sanitize result: %q", got)
	}
}
