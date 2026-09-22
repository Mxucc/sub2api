//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/pkg/billingexpr"
	"github.com/stretchr/testify/require"
)

// varPName is the input-token variable name, used to index tier coefficients.
const varPName = billingexpr.VarP

// testPricingInstant returns a fixed billing instant so tests that consult the
// clock are deterministic.
func testPricingInstant() time.Time {
	return time.Date(2026, 9, 15, 2, 0, 0, 0, time.UTC)
}

// newCatalogTestService builds a BillingService whose catalog holds exactly the
// given entries, so assertions do not depend on the shipped price files.
func newCatalogTestService(t *testing.T, catalogJSON string) *BillingService {
	t.Helper()
	pricingService := &PricingService{cfg: &config.Config{}}
	data, err := pricingService.parsePricingData([]byte(catalogJSON))
	require.NoError(t, err)
	pricingService.pricingData = data
	return NewBillingService(&config.Config{}, pricingService)
}

const catalogTestJSON = `{
	"gpt-5.6-luna": {"litellm_provider": "openai", "mode": "chat",
		"input_cost_per_token": 2e-07, "output_cost_per_token": 1.2e-06,
		"cache_read_input_token_cost": 2e-08},
	"deepseek-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 1.5e-07, "output_cost_per_token": 6e-07,
		"cache_read_input_token_cost": 3e-09},
	"expr-model": {"litellm_provider": "test", "mode": "chat",
		"input_cost_per_token": 3e-06, "output_cost_per_token": 1.5e-05,
		"billing_expr": "len <= 1000 ? tier(\"short\", p * 1 + c * 2) : tier(\"long\", p * 3 + c * 6)"},
	"odd-expr-model": {"litellm_provider": "test", "mode": "chat",
		"input_cost_per_token": 1e-06, "output_cost_per_token": 2e-06,
		"billing_expr": "p * 5 + c * 10"},
	"image-only": {"litellm_provider": "test", "mode": "image",
		"output_cost_per_image": 0.04}
}`

// toViewIndex makes assertions readable regardless of the catalog's sort order.
func toViewIndex(views []ModelPricingView) map[string]ModelPricingView {
	index := make(map[string]ModelPricingView, len(views))
	for _, view := range views {
		index[view.Model] = view
	}
	return index
}

func TestListModelPricingCatalog_IncludesModelsWithoutChannels(t *testing.T) {
	svc := newCatalogTestService(t, catalogTestJSON)

	views := svc.ListModelPricingCatalog()
	require.NotEmpty(t, views)

	// The whole point of the page: a model with no channel and no traffic is
	// still listed, because the catalog is a price source rather than a
	// reflection of what is currently served.
	index := toViewIndex(views)
	for _, model := range []string{"gpt-5.6-luna", "deepseek-flash", "expr-model", "odd-expr-model"} {
		require.Contains(t, index, model, "catalog must list %s", model)
	}
}

func TestListModelPricingCatalog_PerMillionConversion(t *testing.T) {
	svc := newCatalogTestService(t, catalogTestJSON)
	view := toViewIndex(svc.ListModelPricingCatalog())["gpt-5.6-luna"]

	// The page displays per-million rates; the API carries both so a caller never
	// has to guess the scale.
	require.InDelta(t, 0.2, view.InputPricePerMillion, 1e-12)
	require.InDelta(t, 1.2, view.OutputPricePerMillion, 1e-12)
	require.InDelta(t, 0.02, view.CacheReadPricePerMillion, 1e-12)
	require.InDelta(t, view.InputPricePerToken*1e6, view.InputPricePerMillion, 1e-15)
	require.Equal(t, "catalog", view.Source)
	require.Equal(t, "openai", view.Provider)
	require.False(t, view.FallbackOnly)
}

// TestListModelPricingCatalog_ExpressionAndTiers pins the field the price page
// exists for: a model whose price is time- or length-dependent must expose both
// the raw expression and its parsed tiers.
func TestListModelPricingCatalog_ExpressionAndTiers(t *testing.T) {
	svc := newCatalogTestService(t, catalogTestJSON)
	view := toViewIndex(svc.ListModelPricingCatalog())["expr-model"]

	require.NotEmpty(t, view.BillingExpr)
	require.Equal(t, "catalog", view.BillingExprSource)
	require.True(t, view.ExprRecognized)
	require.Len(t, view.Tiers, 2)
	require.Equal(t, "short", view.Tiers[0].Name)
	require.Equal(t, "len <= 1000", view.Tiers[0].Condition)
	require.InDelta(t, 1.0, view.Tiers[0].Coefficients[varPName], 1e-12)
	require.Equal(t, "long", view.Tiers[1].Name)
	require.Empty(t, view.Tiers[1].Condition, "the final else branch has no condition")
}

func TestListModelPricingCatalog_TimeDependentFlag(t *testing.T) {
	svc := newCatalogTestService(t, catalogTestJSON)
	index := toViewIndex(svc.ListModelPricingCatalog())

	// deepseek-flash picks up the built-in time-of-day expression, so the page
	// must flag it as time dependent; the length-tiered model must not.
	require.True(t, index["deepseek-flash"].TimeDependent)
	require.Equal(t, "builtin", index["deepseek-flash"].BillingExprSource)
	require.Len(t, index["deepseek-flash"].Tiers, 2)
	require.False(t, index["expr-model"].TimeDependent)
	require.False(t, index["gpt-5.6-luna"].TimeDependent)
}

// TestListModelPricingCatalog_UnrecognizedExpressionStillReported guards the
// page's fallback contract: an expression the display parser cannot read still
// has to reach the client, with the raw string intact and no tiers invented.
func TestListModelPricingCatalog_UnrecognizedExpressionStillReported(t *testing.T) {
	svc := newCatalogTestService(t, catalogTestJSON)
	view := toViewIndex(svc.ListModelPricingCatalog())["odd-expr-model"]

	require.Equal(t, "p * 5 + c * 10", view.BillingExpr)
	require.False(t, view.ExprRecognized)
	require.Empty(t, view.Tiers)
}

// TestListModelPricingCatalog_IncludesBuiltinOnlyModels covers the union: a model
// the price catalog has never heard of but the compiled-in table prices must
// still appear, flagged so an operator knows which table to edit.
func TestListModelPricingCatalog_IncludesBuiltinOnlyModels(t *testing.T) {
	svc := newCatalogTestService(t, catalogTestJSON)

	var found *ModelPricingView
	for _, view := range svc.ListModelPricingCatalog() {
		if view.Model == "claude-opus-4.5" {
			candidate := view
			found = &candidate
			break
		}
	}
	require.NotNil(t, found, "a built-in-only model must still be listed")
	require.Equal(t, "builtin", found.Source)
	require.True(t, found.FallbackOnly)
	require.Greater(t, found.InputPricePerMillion, 0.0)
}

// TestListModelPricingCatalog_IsSorted makes the API stable: an unsorted map
// walk would reshuffle rows between refreshes.
func TestListModelPricingCatalog_IsSorted(t *testing.T) {
	svc := newCatalogTestService(t, catalogTestJSON)
	views := svc.ListModelPricingCatalog()
	require.NotEmpty(t, views)
	for i := 1; i < len(views); i++ {
		require.LessOrEqual(t, views[i-1].Model, views[i].Model,
			"catalog must be sorted: %q came before %q", views[i-1].Model, views[i].Model)
	}
}

func TestListModelPricingCatalog_NoPricingService(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)
	// A nil pricing service must not panic; the built-in table still yields rows.
	require.NotPanics(t, func() { svc.ListModelPricingCatalog() })
	require.NotEmpty(t, svc.ListModelPricingCatalog())
}

func TestDescribeModelBillingExpr(t *testing.T) {
	svc := newCatalogTestService(t, catalogTestJSON)

	described := svc.DescribeModelBillingExpr("deepseek-flash", testPricingInstant())
	require.NotNil(t, described)
	require.Equal(t, "builtin", described.Source)
	require.NotNil(t, described.Parsed)
	require.True(t, described.Parsed.Recognized)
	require.True(t, described.Parsed.TimeDependent)
	require.Len(t, described.Parsed.Tiers, 2)

	// A model with no expression yields nil, so callers can omit the block
	// entirely rather than render an empty one.
	require.Nil(t, svc.DescribeModelBillingExpr("gpt-5.6-luna", testPricingInstant()))
	// An unknown model must not panic and must yield nil.
	require.Nil(t, svc.DescribeModelBillingExpr("no-such-model-at-all", testPricingInstant()))
}

// TestEffectiveBillingExpr_CatalogWinsOverBuiltin pins the precedence rule the
// price page's "source" column advertises: an operator editing the price table
// must see their expression take effect.
func TestEffectiveBillingExpr_CatalogWinsOverBuiltin(t *testing.T) {
	// The catalog's own expression, with the quotes JSON requires escaped.
	const expr = `tier("flat", p * 1 + c * 2)`
	catalogJSON := `{"deepseek-flash": {"litellm_provider": "deepseek", "mode": "chat",
		"input_cost_per_token": 1.5e-07, "output_cost_per_token": 6e-07,
		"billing_expr": "tier(\"flat\", p * 1 + c * 2)"}}`
	svc := newCatalogTestService(t, catalogJSON)

	view := toViewIndex(svc.ListModelPricingCatalog())["deepseek-flash"]
	require.Equal(t, expr, view.BillingExpr)
	require.Equal(t, "catalog", view.BillingExprSource)
	require.False(t, view.TimeDependent, "the catalog expression has no time condition")
}
