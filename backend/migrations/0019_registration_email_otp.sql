-- Email OTP verification for student registration
CREATE TABLE IF NOT EXISTS email_verification_otps (
    id                  BIGSERIAL PRIMARY KEY,
    email               TEXT NOT NULL,
    otp_hash            TEXT NOT NULL,
    attempts            INT NOT NULL DEFAULT 0,
    verified            BOOLEAN NOT NULL DEFAULT FALSE,
    verification_token  TEXT,
    token_expires_at    TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_verification_otps_email_created
    ON email_verification_otps (email, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_email_verification_otps_verification_token
    ON email_verification_otps (verification_token)
    WHERE verification_token IS NOT NULL;
