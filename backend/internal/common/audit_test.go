package common

import "testing"

func TestSanitizeAuditDetails_StripsSensitiveKeys(t *testing.T) {
	out := sanitizeAuditDetails(map[string]any{
		"email":    "user@example.com",
		"password": "secret",
		"token":    "abc",
	})
	if _, ok := out["password"]; ok {
		t.Fatal("password should be stripped")
	}
	if _, ok := out["token"]; ok {
		t.Fatal("token should be stripped")
	}
	if out["email"] != "user@example.com" {
		t.Fatal("expected email to remain")
	}
}

func TestSplitResource(t *testing.T) {
	typ, id := splitResource("user/42")
	if typ != "user" || id != "42" {
		t.Fatalf("unexpected split: %s %s", typ, id)
	}
}

func TestParseActorUserID(t *testing.T) {
	id := parseActorUserID("99")
	if id == nil || *id != 99 {
		t.Fatal("expected user id 99")
	}
}
