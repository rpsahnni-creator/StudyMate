-- Sprint 5-A: billing schema extensions + plan seeds

ALTER TABLE plans
    ADD COLUMN IF NOT EXISTS slug TEXT,
    ADD COLUMN IF NOT EXISTS price_paise BIGINT,
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'INR',
    ADD COLUMN IF NOT EXISTS billing_interval TEXT NOT NULL DEFAULT 'monthly',
    ADD COLUMN IF NOT EXISTS is_popular BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX IF NOT EXISTS idx_plans_slug ON plans (slug) WHERE slug IS NOT NULL;

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS plan_id BIGINT REFERENCES plans(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS provider_order_id TEXT,
    ADD COLUMN IF NOT EXISTS provider_payment_id TEXT,
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'INR',
    ADD COLUMN IF NOT EXISTS amount_paise BIGINT,
    ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_provider_order
    ON payments (provider, provider_order_id)
    WHERE provider_order_id IS NOT NULL;

ALTER TABLE payment_events
    ALTER COLUMN payment_id DROP NOT NULL;

ALTER TABLE payment_events
    ADD COLUMN IF NOT EXISTS provider_event_id TEXT,
    ADD COLUMN IF NOT EXISTS provider TEXT,
    ADD COLUMN IF NOT EXISTS processed BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS received_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_events_provider_event_id
    ON payment_events (provider_event_id)
    WHERE provider_event_id IS NOT NULL;

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;

DELETE FROM subscriptions a
USING subscriptions b
WHERE a.id < b.id AND a.user_id = b.user_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_user_id_unique ON subscriptions (user_id);

INSERT INTO plans (slug, name, price_monthly, price_paise, currency, billing_interval, scan_limit, features_json, active, is_popular, created_at)
SELECT 'plan_basic_monthly', 'Basic', 199.00, 19900, 'INR', 'monthly', 10,
       '["10 scans/day","Basic quiz","Email support"]'::jsonb, true, false, now()
WHERE NOT EXISTS (SELECT 1 FROM plans WHERE slug = 'plan_basic_monthly');

INSERT INTO plans (slug, name, price_monthly, price_paise, currency, billing_interval, scan_limit, features_json, active, is_popular, created_at)
SELECT 'plan_pro_monthly', 'Pro', 499.00, 49900, 'INR', 'monthly', 0,
       '["Unlimited scans","AI quiz","Priority support","Career goals (coming soon)"]'::jsonb, true, true, now()
WHERE NOT EXISTS (SELECT 1 FROM plans WHERE slug = 'plan_pro_monthly');
