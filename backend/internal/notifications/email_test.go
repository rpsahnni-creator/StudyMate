package notifications

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestVerifyResendSignature_Valid(t *testing.T) {
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("super-secret-signing-key"))
	payload := []byte(`{"type":"email.delivered","email":"user@example.com"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	id := "msg_test123"

	sig := signSvixPayload(secret, id, ts, payload)

	ok := VerifyResendSignature(payload, ResendWebhookHeaders{
		ID:        id,
		Timestamp: ts,
		Signature: "v1," + sig,
	}, secret)
	if !ok {
		t.Fatal("expected valid signature")
	}
}

func TestVerifyResendSignature_InvalidSecret(t *testing.T) {
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("super-secret-signing-key"))
	payload := []byte(`{"type":"email.delivered"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	id := "msg_test123"
	sig := signSvixPayload(secret, id, ts, payload)

	ok := VerifyResendSignature(payload, ResendWebhookHeaders{
		ID:        id,
		Timestamp: ts,
		Signature: "v1," + sig,
	}, "whsec_" + base64.StdEncoding.EncodeToString([]byte("other-secret")))
	if ok {
		t.Fatal("expected invalid signature for wrong secret")
	}
}

func TestVerifyResendSignature_ExpiredTimestamp(t *testing.T) {
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("super-secret-signing-key"))
	payload := []byte(`{"type":"email.delivered"}`)
	ts := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	id := "msg_test123"
	sig := signSvixPayload(secret, id, ts, payload)

	ok := VerifyResendSignature(payload, ResendWebhookHeaders{
		ID:        id,
		Timestamp: ts,
		Signature: "v1," + sig,
	}, secret)
	if ok {
		t.Fatal("expected rejection for expired timestamp")
	}
}

func signSvixPayload(secret, id, timestamp string, payload []byte) string {
	key, err := decodeWebhookSecret(secret)
	if err != nil {
		panic(err)
	}
	signedContent := fmt.Sprintf("%s.%s.%s", id, timestamp, string(payload))
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(signedContent))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
