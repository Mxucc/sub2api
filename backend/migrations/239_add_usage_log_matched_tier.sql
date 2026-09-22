-- Snapshot which declarative billing-expression tier (e.g. "peak"/"off_peak")
-- a request matched, and whether a billing expression priced it at all, so
-- usage history can report how many requests hit each tier without inferring
-- from totals. Empty matched_tier means no expression or no tier() call.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS matched_tier VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS billing_expr_applied BOOLEAN NOT NULL DEFAULT FALSE;
