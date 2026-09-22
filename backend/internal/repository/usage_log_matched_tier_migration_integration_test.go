//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MatchedTierMigrationSuite struct {
	IntegrationDBSuite
}

func TestMatchedTierMigrationSuite(t *testing.T) {
	suite.Run(t, new(MatchedTierMigrationSuite))
}

// TestMigration239_DefaultsOnExistingRows is the important one: the migration
// must apply cleanly to a usage_logs table that already holds data and leave the
// new columns at their declared defaults. The columns are dropped first to
// reproduce a deployment that predates migration 239, then the migration is run
// twice to prove it is idempotent.
func (s *MatchedTierMigrationSuite) TestMigration239_DefaultsOnExistingRows() {
	ctx := context.Background()
	t := s.T()

	user := mustCreateUser(t, s.client, &service.User{Email: "mig-239@test.com"})
	apiKey := mustCreateApiKey(t, s.client, &service.APIKey{UserID: user.ID, Key: "sk-mig-239", Name: "k"})
	account := mustCreateAccount(t, s.client, &service.Account{Name: "acc-mig-239"})

	_, err := s.tx.ExecContext(ctx, `
ALTER TABLE usage_logs
	DROP COLUMN IF EXISTS matched_tier,
	DROP COLUMN IF EXISTS billing_expr_applied
`)
	require.NoError(t, err)

	rows, err := s.tx.QueryContext(ctx, `
INSERT INTO usage_logs (user_id, api_key_id, account_id, request_id, model, created_at)
VALUES ($1, $2, $3, 'migration-239-existing', 'gpt-5', NOW())
RETURNING id
`, user.ID, apiKey.ID, account.ID)
	require.NoError(t, err)
	require.True(t, rows.Next())
	var id int64
	require.NoError(t, rows.Scan(&id))
	require.NoError(t, rows.Close())

	migrationSQL, err := dbmigrations.FS.ReadFile("239_add_usage_log_matched_tier.sql")
	require.NoError(t, err)

	_, err = s.tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)
	_, err = s.tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	rows, err = s.tx.QueryContext(ctx, `
SELECT matched_tier, billing_expr_applied FROM usage_logs WHERE id = $1
`, id)
	require.NoError(t, err)
	require.True(t, rows.Next())
	var matchedTier string
	var billingExprApplied bool
	require.NoError(t, rows.Scan(&matchedTier, &billingExprApplied))
	require.NoError(t, rows.Close())

	require.Equal(t, "", matchedTier)
	require.False(t, billingExprApplied)
}

// TestUsageLogRepository_RoundTripsBillingExpressionFields pins the insert ->
// select round-trip for the two new columns.
func (s *MatchedTierMigrationSuite) TestUsageLogRepository_RoundTripsBillingExpressionFields() {
	ctx := context.Background()
	t := s.T()

	user := mustCreateUser(t, s.client, &service.User{Email: "mig-239-rt@test.com"})
	apiKey := mustCreateApiKey(t, s.client, &service.APIKey{UserID: user.ID, Key: "sk-mig-239-rt", Name: "k"})
	account := mustCreateAccount(t, s.client, &service.Account{Name: "acc-mig-239-rt"})

	repo := newUsageLogRepositoryWithSQL(s.client, s.tx)
	log := &service.UsageLog{
		UserID:             user.ID,
		APIKeyID:           apiKey.ID,
		AccountID:          account.ID,
		RequestID:          "req-mig-239-roundtrip",
		Model:              "gpt-5",
		MatchedTier:        "peak",
		BillingExprApplied: true,
	}
	inserted, err := repo.Create(ctx, log)
	require.NoError(t, err)
	require.True(t, inserted)

	got, err := repo.GetByID(ctx, log.ID)
	require.NoError(t, err)
	require.Equal(t, "peak", got.MatchedTier)
	require.True(t, got.BillingExprApplied)
}
