//go:build unit

package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// newAdminTuningTestService builds a PricingService backed by a real temp
// directory: a catalog file that stands in for the downloaded price table, and an
// override file the admin would edit.
//
// It goes through the real load path rather than poking pricingData directly, so
// the test covers the whole chain a tune travels: file → override merge →
// in-memory catalog → the price the billing service reads.
func newAdminTuningTestService(t *testing.T, catalogJSON string) (*PricingService, string) {
	t.Helper()
	dataDir := t.TempDir()

	catalogPath := filepath.Join(dataDir, "model_pricing.json")
	require.NoError(t, os.WriteFile(catalogPath, []byte(catalogJSON), 0o644))

	overridePath := filepath.Join(dataDir, "model_pricing_overrides.json")

	svc := &PricingService{cfg: &config.Config{}}
	svc.cfg.Pricing.DataDir = dataDir
	svc.cfg.Pricing.FallbackFile = ""
	svc.cfg.Pricing.OverrideFile = overridePath

	require.NoError(t, svc.loadPricingData(catalogPath))
	return svc, overridePath
}

const adminTuningCatalogJSON = `{
	"gpt-5.6-luna": {"litellm_provider": "openai", "mode": "chat",
		"input_cost_per_token": 2e-07, "output_cost_per_token": 1.2e-06,
		"cache_read_input_token_cost": 2e-08},
	"deepseek-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 1.5e-07, "output_cost_per_token": 6e-07}
}`

// TestAdminTuning_TunedPriceReachesBilling is the core promise of the feature: an
// operator edits a price in the admin page and the next request is billed at it,
// with no restart and no release.
func TestAdminTuning_TunedPriceReachesBilling(t *testing.T) {
	svc, _ := newAdminTuningTestService(t, adminTuningCatalogJSON)

	// Before tuning, billing sees the catalogue rate.
	billing := NewBillingService(&config.Config{}, svc)
	before, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 2e-07, before.InputPricePerToken, 1e-15)

	_, err = svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{
		"input_cost_per_token":  5e-06,
		"output_cost_per_token": 2.5e-05,
	})
	require.NoError(t, err)

	after, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 5e-06, after.InputPricePerToken, 1e-15,
		"a tuned price must be what billing reads")
	require.InDelta(t, 2.5e-05, after.OutputPricePerToken, 1e-15)
	// A field the tune did not name keeps the catalogue value: the override is a
	// patch, not a replacement entry.
	require.InDelta(t, 2e-08, after.CacheReadPricePerToken, 1e-15)
}

// TestAdminTuning_NullFieldFallsBackToTheCatalog pins the "clear this field"
// gesture: null removes the patch key, so the catalogue value applies again.
func TestAdminTuning_NullFieldFallsBackToTheCatalog(t *testing.T) {
	svc, _ := newAdminTuningTestService(t, adminTuningCatalogJSON)
	billing := NewBillingService(&config.Config{}, svc)

	_, err := svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{"input_cost_per_token": 9e-06})
	require.NoError(t, err)
	tuned, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 9e-06, tuned.InputPricePerToken, 1e-15)

	_, err = svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{"input_cost_per_token": nil})
	require.NoError(t, err)

	restored, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 2e-07, restored.InputPricePerToken, 1e-15,
		"clearing the patch must fall back to the price table")
}

// TestAdminTuning_DeleteRestoresTheTablePrice pins the "restore price-table price"
// action end to end.
//
// The model is deliberately not a DeepSeek one: DeepSeek's catalogue rate is
// force-overridden in code (the official off-peak card), so it could not show that
// the tune itself had been removed.
func TestAdminTuning_DeleteRestoresTheTablePrice(t *testing.T) {
	svc, _ := newAdminTuningTestService(t, adminTuningCatalogJSON)
	billing := NewBillingService(&config.Config{}, svc)

	_, err := svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{"input_cost_per_token": 9e-06})
	require.NoError(t, err)
	tuned, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 9e-06, tuned.InputPricePerToken, 1e-15)

	removed, err := svc.DeleteOverrideEntry("gpt-5.6-luna")
	require.NoError(t, err)
	require.True(t, removed)

	restored, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 2e-07, restored.InputPricePerToken, 1e-15,
		"deleting the entry must restore the price-table rate")

	// Deleting twice is idempotent: a double click must not error.
	again, err := svc.DeleteOverrideEntry("gpt-5.6-luna")
	require.NoError(t, err)
	require.False(t, again)
}

// TestAdminTuning_ChannelPricingStillWins is the precedence guarantee the feature
// promises in its UI copy: a channel that prices the model explicitly keeps its
// price, because the override layer sits below channel pricing.
func TestAdminTuning_ChannelPricingStillWins(t *testing.T) {
	svc, _ := newAdminTuningTestService(t, adminTuningCatalogJSON)

	_, err := svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{"input_cost_per_token": 5e-06})
	require.NoError(t, err)

	billing := NewBillingService(&config.Config{}, svc)
	withChannel, err := billing.GetModelPricingWithChannel("gpt-5.6-luna", &ChannelModelPricing{
		BillingMode: BillingModeToken,
		InputPrice:  float64Ptr(7e-06),
	})
	require.NoError(t, err)
	require.InDelta(t, 7e-06, withChannel.InputPricePerToken, 1e-15,
		"an explicit channel price must beat the admin tune")
}

// TestAdminTuning_RefreshKeepsAndOverwritesTunedEntries covers the 获取最新价格
// dialog's two branches at the level that matters: what billing then charges.
func TestAdminTuning_RefreshKeepsAndOverwritesTunedEntries(t *testing.T) {
	svc, _ := newAdminTuningTestService(t, adminTuningCatalogJSON)

	_, err := svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{"input_cost_per_token": 5e-06})
	require.NoError(t, err)

	// "Keep my tunings": the tuned model keeps its price, and a model the
	// catalogue serves but the file did not mention gets an entry.
	result, err := svc.RefreshOverrides(false)
	require.NoError(t, err)
	require.GreaterOrEqual(t, result.Added, 1, "an untuned catalogued model must get an entry")
	require.GreaterOrEqual(t, result.Kept, 1, "the tuned entry must be kept")

	billing := NewBillingService(&config.Config{}, svc)
	kept, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 5e-06, kept.InputPricePerToken, 1e-15,
		"refresh with keep-tunings must not touch a tuned price")

	// "Overwrite everything": the tuned price is replaced by the catalogue rate.
	_, err = svc.RefreshOverrides(true)
	require.NoError(t, err)

	overwritten, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 2e-07, overwritten.InputPricePerToken, 1e-15,
		"refresh with overwrite must reset the tuned price to the catalogue rate")
}

// TestAdminTuning_UnknownFieldIsRejectedAndChangesNothing guards the failure mode a
// typo would otherwise create: a rejected write must leave billing untouched.
func TestAdminTuning_UnknownFieldIsRejectedAndChangesNothing(t *testing.T) {
	svc, overridePath := newAdminTuningTestService(t, adminTuningCatalogJSON)
	billing := NewBillingService(&config.Config{}, svc)

	_, err := svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{"input_cost": 5e-06})
	require.Error(t, err)
	require.Contains(t, err.Error(), "input_cost", "the error must name the offending field")

	_, statErr := os.Stat(overridePath)
	require.True(t, os.IsNotExist(statErr), "a rejected write must not create the file")

	unchanged, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.InDelta(t, 2e-07, unchanged.InputPricePerToken, 1e-15)
}

// TestAdminTuning_BillingExprTakesOverACatalogModel pins the expression route: a
// catalogue model can be moved onto a declarative price from the admin page, and
// a bad expression is refused.
func TestAdminTuning_BillingExprTakesOverACatalogModel(t *testing.T) {
	svc, _ := newAdminTuningTestService(t, adminTuningCatalogJSON)
	billing := NewBillingService(&config.Config{}, svc)

	const expression = `tier("flat", p * 1 + c * 2)`
	_, err := svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{"billing_expr": expression})
	require.NoError(t, err)

	tuned, err := billing.GetModelPricing("gpt-5.6-luna")
	require.NoError(t, err)
	require.Equal(t, expression, tuned.BillingExpr,
		"the expression must reach the billing service through the override layer")

	_, err = svc.SetOverrideEntry("gpt-5.6-luna", map[string]any{"billing_expr": "tier("})
	require.Error(t, err, "an uncompilable expression must be refused")
}
