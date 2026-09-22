package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMigration239AddsMatchedTierAndBillingExprApplied guards the embedded SQL
// file's shape without a database: the migration must target usage_logs and add
// both billing-expression snapshot columns idempotently (so re-running it on a
// deployment that already applied it is a no-op).
func TestMigration239AddsMatchedTierAndBillingExprApplied(t *testing.T) {
	content, err := FS.ReadFile("239_add_usage_log_matched_tier.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ALTER TABLE usage_logs")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS matched_tier VARCHAR(32) NOT NULL DEFAULT ''")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS billing_expr_applied BOOLEAN NOT NULL DEFAULT FALSE")
}
