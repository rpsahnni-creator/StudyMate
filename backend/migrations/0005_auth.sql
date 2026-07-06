-- Migration 0005: Add authentication-related columns to users table
-- This migration adds email_verified column needed for auth module

ALTER TABLE users 
ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- Create unique index on email for efficient lookup
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Create index on status for filtering active users
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
